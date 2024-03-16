package main

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	pgm "github.com/TomasMen/go-pgm"
)

type LaplacianImage struct {
    Width int
    Height int
    Pixels [][]int
} 

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage:", os.Args[0], "stack.txt")
        fmt.Println("stack.txt must be a text file containing the paths to the images to be combined, one per line. Either a relative path or an absolute path which will be looked for under the \"./stack/\" directory.")
        os.Exit(1)
    }

    fileName := os.Args[1]
    filetype := fileName[len(fileName)-4:]
    if !(filetype == ".txt") {
        fmt.Println("Error: File provided did not end in .txt")
        fmt.Println("Usage:", os.Args[0], "stack.txt")
        fmt.Println("stack.txt must be a text file containing the paths to the images to be combined, one per line. Either a relative path or an absolute path which will be looked for under the \"./stack/\" directory.")
        os.Exit(1)
    }

    file, err := os.Open(fileName)
    if err != nil {
        fmt.Println("Error:", err)
        os.Exit(1)
    }
    defer file.Close() 

    scanner := bufio.NewScanner(file)
    imagePaths := make([]string, 0)
    for scanner.Scan() {
        imagePaths = append(imagePaths, scanner.Text())
    }

    imageStack := make([]pgm.PGMImage, 0)
    for _, imagePath := range imagePaths {
        // Use relative path
        if !filepath.IsAbs(imagePath) {
            image, err := pgm.ReadPGM(imagePath)
            if err == nil {
                imageStack = append(imageStack, *image)
                continue
            }
        }

        // Try in current directory
        if _, err := os.Stat(imagePath); err == nil {
            image, err := pgm.ReadPGM(imagePath)
            if err == nil {
                imageStack = append(imageStack, *image)
                continue
            }
        }

        // Check under the "./stack/" directory
        imagePath = filepath.Join("./stack/", imagePath)
        image, err := pgm.ReadPGM(imagePath)
        if err != nil {
            fmt.Printf("Error loading image '%s': %v\n", imagePath, err)
            continue
        }

        imageStack = append(imageStack, *image)
    }

    if len(imageStack) < 2 {
        if len(imageStack) == 0 {
            fmt.Printf("Error: No valid images were found in the provided text file '%s'\n", fileName) 
        } else {
            fmt.Printf("Error: Not enough valid images were found in the provided text file '%s', only %d files were found, atleast 2 are required.\n", fileName, len(imageStack))
        }
        fmt.Println("Please ensure that the file contains valid image paths, one per line.")
        fmt.Println("The image paths can be either relative or absolute.")
        fmt.Println("Absolute paths will be searched for in the current directory and under the \"./stack/\" directory.")
        os.Exit(1)
    }

    width := imageStack[0].Width
    height := imageStack[0].Height
    for i := 1; i < len(imageStack); i++ {
        if imageStack[i].Width != width || imageStack[i].Height != height {
            fmt.Printf("Error: Image dimensions mismatch. All images must have the same width and height.\n")
            fmt.Printf("Image '%s' has dimensions %dx%d, expected %dx%d\n", imagePaths[i], imageStack[i].Width, imageStack[i].Height, width, height)
            os.Exit(1)
        }
    }

    threshold := 40
    divisor := float64(2)
    maxDivisionsWidth := math.Floor(math.Log(float64(width)/float64(threshold)) / math.Log(divisor))
    maxDivisionsHeight := math.Floor(math.Log(float64(height)/float64(threshold)) / math.Log(divisor))
    maxDivisions := int(math.Min(maxDivisionsWidth, maxDivisionsHeight))

    pyramidStack := make([][]pgm.PGMImage, len(imageStack))
    gaussianPyramidStack := make([][]pgm.PGMImage, len(imageStack))
    laplacianPyramidStack := make([][]LaplacianImage, len(imageStack))
    for idx, image := range imageStack {
        pyramidStack[idx] = createImagePyramid(image, maxDivisions, divisor)
        gaussianPyramidStack[idx] = createGaussianPyramid(pyramidStack[idx])
        laplacianPyramidStack[idx], err = createLaplacianPyramid(pyramidStack[idx], gaussianPyramidStack[idx])
        if err != nil {
            fmt.Println("Error: An error occurred while creating the laplacian pyramid:", err)
            os.Exit(1)
        }
    }
    coarsestImageStack := make([]LaplacianImage, len(pyramidStack))
    for idx, pyramid := range pyramidStack {
        coarsestImageStack[idx] = pgmToLaplacianImageType(pyramid[len(pyramidStack[0])-1])
    }

    meanImage, err := meanImages(coarsestImageStack)
    if err != nil {
        fmt.Println("Error: Could not calculate mean of coarsest image:", err)
        os.Exit(1)
    }

    maxLaplacianPyramid := createMaxLaplacianPyramid(laplacianPyramidStack)

    // fmt.Printf("Mean image dimensions: %dx%d\n", meanImage.Width, meanImage.Height)
    // fmt.Printf("maxLaplacianPyramid height: %d\n", len(maxLaplacianPyramid))
    // fmt.Printf("Base pyramid height: %d\n", len(pyramidStack[0]))
    // fmt.Printf("Biggest image in base pyramid dimensions: %dx%d\n", pyramidStack[0][3].Width, pyramidStack[0][3].Height)
    // fmt.Printf("Maximum divisons: %d\n", maxDivisions)

    reconstructedImage := reconstructImage(*meanImage, maxLaplacianPyramid)
    reconstructedImageClamped := clampImage(reconstructedImage, 0, 255)
    err = pgm.WritePGM(*reconstructedImageClamped, "result.pgm")
    if err != nil {
        fmt.Println("Error: Failed to write final pgm file:", err)
        os.Exit(1)
    }
}

func clampImage(image LaplacianImage, minValue, maxValue uint8) *pgm.PGMImage {
    pixelArray := make([][]uint8, image.Height)
    for row := range pixelArray {
        pixelArray[row] = make([]uint8, image.Width)
    }

    for y := 0; y < image.Height; y++ {
        for x := 0; x < image.Width; x++ {
            if image.Pixels[y][x] > int(maxValue) {
                pixelArray[y][x] = maxValue
            } else if image.Pixels[y][x] < int(minValue) {
                pixelArray[y][x] = minValue
            } else {
                pixelArray[y][x] = uint8(image.Pixels[y][x])
            }
        }
    }

    return & pgm.PGMImage {
        Width: image.Width,
        Height: image.Height,
        MaxVal: 255,
        Pixels: pixelArray,
    }
}

func reconstructImage(seedImage LaplacianImage, maxPyramid []LaplacianImage) LaplacianImage {
    reconstructedImage := LaplacianImage {
        Width: seedImage.Width,
        Height: seedImage.Height,
        Pixels: seedImage.Pixels, 
    }

    for i := len(maxPyramid)-1; i >= 0; i-- {
        width := maxPyramid[i].Width
        height := maxPyramid[i].Height
        upscaledImage := resizeImageLaplacian(reconstructedImage, width, height)
        result, err := addImagesLaplacian(upscaledImage, maxPyramid[i])
        if err != nil {
            fmt.Println("Error: Could not reconstruct image:", err)
            os.Exit(1)
        }
        reconstructedImage = *result
    }
     
    return reconstructedImage
}

func addImagesLaplacian(image1 LaplacianImage, image2 LaplacianImage) (*LaplacianImage, error) {
    if image1.Height != image2.Height || image1.Width != image2.Width {
        return nil, errors.New("Error: Image dimension mismatch, cannot add two images of differing sizes")
    }
    
     newImage := &LaplacianImage{
        Width:  image1.Width,
        Height: image1.Height,
        Pixels: make([][]int, image1.Height),
    }

    for y := 0; y < image1.Height; y++ {
        newImage.Pixels[y] = make([]int, image1.Width)
        for x := 0; x < image1.Width; x++ {
            newImage.Pixels[y][x] = image1.Pixels[y][x] + image2.Pixels[y][x]
        }
    }

    return newImage, nil
}

func pgmToLaplacianImageType(image pgm.PGMImage) LaplacianImage {
    laplacianImage := LaplacianImage {
        Width: image.Width,
        Height: image.Height,
        Pixels: make([][]int, image.Height),
    }

    for row := range laplacianImage.Pixels {
        laplacianImage.Pixels[row] = make([]int, image.Width)
    }

    for y := 0; y < image.Height; y++ {
        for x := 0; x < image.Width; x++ {
            laplacianImage.Pixels[y][x] = int(image.Pixels[y][x])
        }
    }
    return laplacianImage
}

func meanImages(images []LaplacianImage) (*LaplacianImage, error) {
    numOfImages := len(images)

    width := images[0].Width
    height := images[0].Height
    for idx := 1; idx < numOfImages; idx++ {
        if images[idx].Height != height || images[idx].Width != width {
            return nil, errors.New("Image size mismatch, cannot calculate mean of different sized images")
        }
    }

    meanImageArray := make([][]int, height)
    for row := range meanImageArray {
        meanImageArray[row] = make([]int, width)
    }

    for y := 0; y < height; y++ {
        for x := 0; x < width; x++ {
            var total int = 0
            for idx := range images {
                total += images[idx].Pixels[y][x]
            }
            meanImageArray[y][x] = int( float64(total) / float64(numOfImages) )
        }
    }

    return &LaplacianImage {
        Width: width,
        Height: height,
        Pixels: meanImageArray,
    }, nil
}

func createMaxLaplacianPyramid(laplacianPyramidStack [][]LaplacianImage) []LaplacianImage {
    maxLaplacianPyramid := make([]LaplacianImage, len(laplacianPyramidStack[0]))
    for pyramidTier := range laplacianPyramidStack[0] {
        width := laplacianPyramidStack[0][pyramidTier].Width
        height := laplacianPyramidStack[0][pyramidTier].Height

        maxLaplacianImage := LaplacianImage {
            Height: height,
            Width: width,
            Pixels: make([][]int, height),
        }

        for row := range maxLaplacianImage.Pixels {
            maxLaplacianImage.Pixels[row] = make([]int, maxLaplacianImage.Width)
        }

        for y := 0; y < height; y++ {
            for x:= 0; x < width; x++ {
                maxResponse := 0

                for _, pyramid := range laplacianPyramidStack {
                    value := pyramid[pyramidTier].Pixels[y][x]
                    if math.Abs(float64(value)) > math.Abs(float64(maxResponse)) {
                        maxResponse = value
                    }
                }

                maxLaplacianImage.Pixels[y][x] = maxResponse
            }
        }

        maxLaplacianPyramid[pyramidTier] = maxLaplacianImage
    }

    return maxLaplacianPyramid
}

func createLaplacianPyramid(pyramid []pgm.PGMImage, gaussianPyramid []pgm.PGMImage) ([]LaplacianImage, error) {
    laplacianPyramid := make([]LaplacianImage, len(gaussianPyramid))
    for idx := range gaussianPyramid {
        laplacian, err := subtractImages(pyramid[idx].Pixels, gaussianPyramid[idx].Pixels)
        if err != nil {
            return nil, err
        }
        laplacianPyramid[idx] = LaplacianImage {
            Width: pyramid[idx].Width,
            Height: pyramid[idx].Height,
            Pixels: laplacian,
        }
    }
    return laplacianPyramid, nil
}

func subtractImages(image1, image2 [][]uint8) ([][]int, error) {
    if len(image1) != len(image2) || len(image1[0]) != len(image2[0]) {
        return nil, errors.New("Error: Image dimension mismatch, cannot subtract two images with differing sizes")
    }

    resultingImage := make([][]int, len(image1))
    for row := range resultingImage {
        resultingImage[row] = make([]int, len(image1[0]))
    }

    for y := 0; y < len(image1); y++ {
        for x := 0; x < len(image1[0]); x++ {
            resultingImage[y][x] = int(image1[y][x]) - int(image2[y][x])
        }
    }

    return resultingImage, nil
}

func createImagePyramid(image pgm.PGMImage, numOfDivisions int, divisor float64) []pgm.PGMImage {
    pyramid := make([]pgm.PGMImage, numOfDivisions+1)
    pyramid[0] = image

    for i := 1; i<len(pyramid); i++ {
        divisorScaled := math.Pow(divisor, float64(i))
        targetWidth := int( float64(image.Width) / divisorScaled )
        targetHeight := int( float64(image.Height) / divisorScaled )
        pyramid[i] = resizeImage(image, targetWidth, targetHeight)
    }

    return pyramid
}

func createGaussianPyramid(pyramid []pgm.PGMImage) []pgm.PGMImage {
    gaussianPyramid := make([]pgm.PGMImage, len(pyramid)-1)
    for i := range gaussianPyramid {
        gaussianPyramid[i] = resizeImage(pyramid[i+1], pyramid[i].Width, pyramid[i].Height)
    }
    return gaussianPyramid
}

func resizeImage(image pgm.PGMImage, targetWidth, targetHeight int) pgm.PGMImage {
    startingWidth := image.Width
    startingHeight := image.Height

    widthRatio := float64(startingWidth-1) / float64(targetWidth-1)
    heightRatio := float64(startingHeight-1) / float64(targetHeight-1)

    newImage := make([][]uint8, targetHeight)
    for row := range newImage {
        newImage[row] = make([]uint8, targetWidth) 
    }

    for y := 0; y < targetHeight; y++ {
        for x:= 0; x < targetWidth; x++ {
            newImage[y][x] = interpolateLinear(image, float64(x)*widthRatio, float64(y)*heightRatio)
        }
    }
    newPGMImage := &pgm.PGMImage {
        Width: targetWidth,
        Height: targetHeight,
        MaxVal: 255,
        Pixels: newImage,
    }

    return *newPGMImage
}

func resizeImageLaplacian(image LaplacianImage, targetWidth, targetHeight int) LaplacianImage {
    startingWidth := image.Width
    startingHeight := image.Height

    widthRatio := float64(startingWidth-1) / float64(targetWidth-1)
    heightRatio := float64(startingHeight-1) / float64(targetHeight-1)

    newImage := make([][]int, targetHeight)
    for row := range newImage {
        newImage[row] = make([]int, targetWidth) 
    }

    for y := 0; y < targetHeight; y++ {
        for x:= 0; x < targetWidth; x++ {
            newImage[y][x] = interpolateLinearLaplacian(image, float64(x)*widthRatio, float64(y)*heightRatio)
        }
    }
    newPGMImage := &LaplacianImage {
        Width: targetWidth,
        Height: targetHeight,
        Pixels: newImage,
    }

    return *newPGMImage
}


func interpolateLinear(image pgm.PGMImage, x, y float64) uint8 {
    x = math.Max(0, math.Min(x, float64(image.Width-1)))
    y = math.Max(0, math.Min(y, float64(image.Height-1)))

    minX := math.Floor(x)
    maxX := math.Ceil(x)
    minY := math.Floor(y)
    maxY := math.Ceil(y)

    dx := x - minX
    dy := y - minY

    // REMINDER TO SELF: Make sure to double check all the data types when doing any arithmetic ;(
    // (Took me 30 mins to find that I was subtracting uint8's causing the values to be out of range)
    topLeftVal := int(image.Pixels[int(minY)][int(minX)])
    topRightVal := int(image.Pixels[int(minY)][int(maxX)])
    bottomLeftVal := int(image.Pixels[int(maxY)][int(minX)])
    bottomRightVal := int(image.Pixels[int(maxY)][int(maxX)])

    topVal := float64(topLeftVal) + float64(topRightVal-topLeftVal)*dx
    bottomVal := float64(bottomLeftVal) + float64(bottomRightVal-bottomLeftVal)*dx

    value := topVal + (bottomVal-topVal)*dy

    if value > 255 || value < 0 {
        fmt.Printf("Warning: Interpolated value %.2f is outside the valid range [0, 255]\n", value)
        // fmt.Printf("MinX: %f MinY: %f MaxX: %f MaxY: %f\n", minX, minY, maxX, maxY)
        // fmt.Printf("tl: %d tr: %d bl: %d br: %d\n", topLeftVal, topRightVal, bottomLeftVal, bottomRightVal)
        // fmt.Printf("dx: %f dy: %f\n", dx, dy)
        // fmt.Printf("topDiff: %f scaledTopDiff: %f\n", topDiff, scaledTopDiff)
        // fmt.Printf("topVal: %f bottomVal: %f\n", topVal, bottomVal) 
        // fmt.Printf("Coordinates: %f,%f\n", x, y)
        // fmt.Printf("Image size %dx%d\n", image.Width, image.Height)
        if value < 0 {
            value = 0
        } else {
            value = 255
        }
    }
    return uint8(value)
}


func interpolateLinearLaplacian(image LaplacianImage, x, y float64) int {
    width := len(image.Pixels[0])
    height := len(image.Pixels)
    x = math.Max(0, math.Min(x, float64(width-1)))
    y = math.Max(0, math.Min(y, float64(height-1)))
    // fmt.Printf("Alleged dimensions: %dx%d, actual dimensions: %dx%d \n", image.Width, image.Height, len(image.Pixels[0]), len(image.Pixels))

    minX := math.Floor(x)
    maxX := math.Ceil(x)
    minY := math.Floor(y)
    maxY := math.Ceil(y)

    dx := x - minX
    dy := y - minY

    // REMINDER TO SELF: Make sure to double check all the data types when doing any arithmetic ;(
    // (Took me 30 mins to find that I was subtracting uint8's causing the values to be out of range)
    topLeftVal := int(image.Pixels[int(minY)][int(minX)])
    topRightVal := int(image.Pixels[int(minY)][int(maxX)])
    bottomLeftVal := int(image.Pixels[int(maxY)][int(minX)])
    bottomRightVal := int(image.Pixels[int(maxY)][int(maxX)])

    topVal := float64(topLeftVal) + float64(topRightVal-topLeftVal)*dx
    bottomVal := float64(bottomLeftVal) + float64(bottomRightVal-bottomLeftVal)*dx

    value := topVal + (bottomVal-topVal)*dy

    return int(value)
}

