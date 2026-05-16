Usage
-----

This package provides a small, stable public wrapper around the internal CloudFront signer.

Example:

```go
package main

import (
    "fmt"
    "time"
    "io/ioutil"

    "github.com/brunojet/go-infra-adapters/pkg/cloudfront/signer"
)

func main() {
    pem, _ := ioutil.ReadFile("/path/to/private.pem")
    s, err := signer.NewSignerFromPEM(pem, "K2JCJMDEHXQW5F")
    if err != nil {
        panic(err)
    }
    signed, err := s.SignURLCanned("https://d111111abcdef8.cloudfront.net/images/image.jpg", time.Now().Add(1*time.Hour))
    if err != nil {
        panic(err)
    }
    fmt.Println(signed)
}
```

Notes
-----
- The implementation delegates to the internal signer located at `internal/cloudfront/signer`.
- `NewSignerFromPEM` accepts a PEM encoded RSA private key in PKCS#1 or PKCS#8 format.
