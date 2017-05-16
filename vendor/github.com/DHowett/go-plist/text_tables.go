package plist

// Bitmap of characters that must be inside a quoted string
// when written to an old-style property list
// Low bits represent lower characters, and each uint64 represents 64 characters.
var gsQuotable = [4]uint64{
	0x78001385ffffffff,
	0xa800000138000000,
	0xffffffffffffffff,
	0xffffffffffffffff,
}

// 7f instead of 3f in the top line: CFOldStylePlist.c says . is valid, but they quote it.
var osQuotable = [4]uint64{
	0xf4007f6fffffffff,
	0xf8000001f8000001,
	0xffffffffffffffff,
	0xffffffffffffffff,
}

var whitespace = [4]uint64{
	0x0000000100003f00,
	0x0000000000000000,
	0x0000000000000000,
	0x0000000000000000,
}
