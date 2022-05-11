# cv-md5-collisions

## Hash collisions

The MD5 collisions are **Identical Prefix Collision**, where both files start out with the data, but have collision
blocks which differ slightly. The specific attack that is being applied is called **UniColl**. The cool part about this
particular attack is that:
* prefix is part of the collision block and no padding is required
* the differences are predictable and quite useful:
    * 10th byte of prefix += 1
    * 10th byte of second block -= 1

## JFIF file format

The JFIF format consists of sections that start with a marker of the form `FF xx`, where `xx` denotes the type of
section. Some common sections are

* `D8` - `SOI` section (start of image)
* `E0` - `APP0` section
* `DA` - `SOS` section (start of scan, image data)
* `FE` - `COM` section (comments)
* `D9` - `EOI` section (end of image)

Some sections like `APP0`, `SOS` and more importantly `COM` have two bytes (a big endian `uint16_t`) which denotes the
size of the section. These two bytes are the ones that are being targetted by the collision attack described above.

## JFIF collisions

The main idea behind JFIF collisions is that a comment section is placed in such a way that the MSB byte of the length
`uint16_t` is the 10th byte of the prefix, which gets incremented by 1. The result is two blobs which hash to the same
digest, but interpreted differently by JFIF parsers.

In one blob, the comment section is 0x100 bytes long (the short comment), whereas in the second blob the comment section
is 0x200 bytes long (the long comment). As a result, there is around 0x100 bytes of data which in one blob is parsed,
whereas in the other blob is considered part of the comment.

In the blob where the data is parsed, it is set to yet another comment where the length of the comment covers all other
sections in the JFIF, such that they are not interpreted. This is done for each part, for a total of 16 parts (because
there are 16 hex digits).

## Steps

### 1. Created the original images

The images under `collisions/white on blue/original` were originally created in GIMP by hand. This step is nothing
special.

### 2. Split images into JFIF sections

The `split` util was used to generate individual files of each JFIF section. The command used was

```sh
cv-md5-utils/split -logtostderr -path image.jpeg
```

### 3. Generate the collisions

#### 3.1. Prepare the collision prefix

TODO add here the image collision step 1

##### 3.1.1. Generate the PDF prefix

This file is created manually by means of `dd`. It starts with something like this:

```pdf
%PDF-1.5
% If you are this curious about how the md5 magic works
% then you should consider inviting me for an interview ;)
% ..................................................................
1 0 obj
<< /Type /XObject /Subtype /Image /Width 35 /Height 51 /BitsPerComponent 8 /Length 57709 /ColorSpace /DeviceRGB /Filter /DCTDecode >>
stream
```

and is followed by more binary data and/or other object declarations. The snippet above is the exact PDF prefix for part
1 of nibble 1 only. For example, part 2 of nibble 1 contains some additional binary data, and part 1 of nibble 2
contains an extra object declaration.

There are several things here to be noted:

1. The comment right after `%PDF-1.5` is usually some stuff inserted by `lualatex` (or maybe `pdflatex`?). The actual
   value depends a bit on which version of `lualatex` was used. The version from Debian 10 was `%<d0><d4><c5><d8>`, but
   then in Debian 11 was `%<cc><d5><c1><d4><c5><d8><d0><c4><c6>`. In the above snippet, that comment is replaced
   completely with a specific message.
2. The length of the padding is chosen arbitrarily and there is no specific reasoning. The idea is to have some content
   to play around with when removing the `lualatex` generated comment or other such things, while maintaining the
   offsets of other objects in the file (e.g. `startxref` and further).
3. The length of the stream (i.e. `57709` in the example above) is tied to the size of the image. However, in the
   beginning there was no way to know what the exact size of the image would be. As such, the first set of collisions
   had an incorrect prefix, but a correct size. After the initial set of collisions, I fixed the prefix and redid the
   collisions with the correct prefix to generate images with a matching size.

##### 3.1.2. Craft the collision prefix

The collision prefix is directly fed to the `poc_no.sh` script which computes the **UniColl** blocks.

If part 1 is being generated, then the collision prefix starts with the aforementioned PDF prefix appended with the
common JFIF segments SOI and APP0. Otherwise, the collision prefix for all other parts is (TODO which file from the
previous part).

The next step is to append a comment section which pads the previous data to a length of 7 mod 64. The reason for 7 is
that, after appending the beginning of yet another comment section, the MSB byte of the `uint16_t` length field will be
on position 10 of the collision block.

Lastly, the starting sequence of a comment section is added, meaning bytes `0xff, 0xfe, 0x01, 0x00`.
