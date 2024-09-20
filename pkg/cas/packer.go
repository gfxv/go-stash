package cas

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
)

type PackFunc func([]byte) []byte
type UnpackFunc func([]byte) ([]byte, error)

// TODO: add levels of compression

func ZLibPack(data []byte) []byte {
	var buff bytes.Buffer
	w := zlib.NewWriter(&buff)
	w.Write(data)
	w.Close()
	return buff.Bytes()
}

func ZLibUnpack(data []byte) ([]byte, error) {
	const op = "cas.packer.ZLibUnpack"

	b := bytes.NewReader(data)
	r, err := zlib.NewReader(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var result bytes.Buffer
	if _, err = io.Copy(&result, r); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return result.Bytes(), nil
}
