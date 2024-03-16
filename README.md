# Go-PGM File Reader

This Go package provides functionality to read Portable Gray Map (PGM) image files in the P5 (raw) format.
It is not a complete package and it was intended to explore how images are read byte by byte, and also the pgm format.

## What it does (Features):

- Reads PGM files with the .pgm extension
- Supports P5 (raw) format PGM images
- Reads the image dimensions (width and height)
- Reads the maximum pixel value (up to 255)
- Stores the pixel intensities in a 2D slice of uint8 values

## What it doesn't do (Limitations):

- This package currently only supports reading P5 (raw) format PGM images
- The maximum pixel value is limited to 255 (uint8 values)

## Installation

To use this package in your Go project, you can install it using the following command:

go get github.com/TomasMen/go-pgm

Here's an example of how to use the gopgm package to read a PGM image file:

```go

package main

import (
    "fmt"
    "github.com/yourusername/gopgm"
)

func main() {
    filePath := "path/to/your/image.pgm"
    image, err := gopgm.ReadPGM(filePath)
    if err != nil {
        fmt.Println("Error:", err)
        return
    }

    fmt.Println("Image width:", image.Width)
    fmt.Println("Image height:", image.Height)
    fmt.Println("Maximum pixel value:", image.MaxVal)

    // Access pixel intensities
    for y := 0; y < image.Height; y++ {
        for x := 0; x < image.Width; x++ {
            pixelValue := image.Pixels[y][x]
            fmt.Printf("Pixel (%d, %d): %d\n", x, y, pixelValue)
        }
    }
}
```

## Error Handling

The ReadPGM function returns an error if any of the following conditions are met:

- Invalid file name (file extension must be .pgm)
- File type is not P5 (raw) format
- Invalid magic number in the PGM header
- Width or height values are not valid ASCII numbers
- Maximum pixel value is not a valid ASCII number or exceeds 255
- End of file is reached before all pixel intensities are read

## Contributing

Contributions to this package are welcome! If you find any issues or have suggestions for improvements, please open an issue or submit a pull request on the GitHub repository.

##License

This package is licensed under the MIT License.

