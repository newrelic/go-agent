package internal

import (
	"testing"
)

type compressEncodeTestcase struct {
	decoded string
	encoded string
}

var testcases = [...]compressEncodeTestcase{
	{decoded: "compress me", encoded: "eJxKzs8tKEotLlbITQUEAAD//xsdBF8="},
	{decoded: "zipzipzipzipzipzipzipzipzipzipzipzipzipzipzipzipzipzipzip" +
		"zipzipzipzipzipzipzipzipzipzipzipzipzipzipzipzipzipzipzip",
		encoded: "eJyqyiygMwIEAAD//0/+MlM="},
}

func TestCompressEncode(t *testing.T) {
	for _, tc := range testcases {
		encoded, err := compressEncode([]byte(tc.decoded))
		if nil != err {
			t.Fatal(err)
		}
		if encoded != tc.encoded {
			t.Fatalf("expected=%s got=%s", tc.encoded, encoded)
		}
		decoded, err := uncompressDecode(encoded)
		if nil != err {
			t.Fatal(err)
		}
		if string(decoded) != tc.decoded {
			t.Fatalf("expected=%s got=%s", tc.decoded, string(decoded))
		}
	}
}
