package downloader

type BitField []byte

// HasPiece 检查某一位是否置1
func (bf BitField) HasPiece(index int) bool {
	byteIndex := index / 8 //获取是在第几个字节
	offset := index % 8    //获取偏移量
	return bf[byteIndex]>>(7-offset)&1 != 0
}

func (bf BitField) SetPiece(index int) {
	byteIndex := index / 8 //获取是在第几个字节
	offset := index % 8    //获取偏移量
	bf[byteIndex] |= 1 << (7 - offset)
}
