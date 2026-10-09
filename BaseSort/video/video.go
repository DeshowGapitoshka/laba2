package video

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

type Header struct {
	smesh      uint32
	width      int
	height     int
	colorDepth uint16
}

var sandwitchGaps = []int{
	1, 5, 19, 41, 109, 209, 505, 929, 2161, 3905,
	8929, 16001, 36289, 64769, 146305, 260609,
	587521, 1045505, 2354689, 4196353,
}

func ReadBMP(path string) (*Header, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Println(err)
		return nil, nil, fmt.Errorf("Произошла ошибка при чтении файла")
	}

	// file_header
	signature := binary.LittleEndian.Uint16(data[:2])
	if signature != 0x4D42 {
		return nil, nil, fmt.Errorf("Не BMP файл")
	}
	// fileSize := binary.LittleEndian.Uint32(data[2:6])
	smesh := binary.LittleEndian.Uint32(data[10:14])
	// info_header
	width := binary.LittleEndian.Uint32(data[18:22])
	height := binary.LittleEndian.Uint32(data[22:26])
	colorDepth := binary.LittleEndian.Uint16(data[28:30])
	if colorDepth != 24 {
		return nil, nil, fmt.Errorf("Должно быть 24 бита, получено %d", colorDepth)
	}
	return &Header{smesh: smesh, width: int(width), height: int(height), colorDepth: colorDepth}, data, nil
}

func ReadBGR(structure *Header, data []byte) []byte {
	stride := ((structure.width*3 + 3) / 4) * 4 // Округление вверх до кратного 4. Stride включает в себя пустые байты.
	pixels := make([]byte, structure.width*structure.height*3)
	for i := 0; i < structure.height; i++ {
		srcStart := int(structure.smesh) + i*stride // с пустыми байтами
		pixelsStart := i * structure.width * 3
		copy(pixels[pixelsStart:pixelsStart+structure.width*3], data[srcStart:srcStart+structure.width*3]) // копирует уже без пустых байтов
	}
	return pixels
}

func ReadKeys(path string) (error, []uint32) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Println(err)
		return fmt.Errorf("Ошибка чтения файла"), nil
	}
	if len(data) == 0 {
		return fmt.Errorf("Пустой файл ключей"), nil
	}

	tok := strings.Fields(string(data)) // Переделываем в массив строк
	count, err := strconv.Atoi(tok[0])
	if err != nil {
		log.Println(err)
		return fmt.Errorf("Невалидный формат данных в документе с ключами"), nil
	}
	if count != len(tok[1:]) {
		return fmt.Errorf("Ожидалось %d ключей, получено %d", count, len(tok[1:])), nil
	}
	keys := make([]uint32, len(tok[1:]))
	for i := 0; i < len(tok[1:]); i++ {
		temp, err := strconv.Atoi(tok[1+i])
		if err != nil {
			log.Println(err)
			return fmt.Errorf("Ошибка перевода строки в число"), nil
		}
		keys[i] = uint32(temp)
	}
	return nil, keys
}

func swap(keys []uint32, pixel []byte, i, j int) {
	keys[i], keys[j] = keys[j], keys[i]
	a, b := i*3, j*3
	pixel[a], pixel[b] = pixel[b], pixel[a]
	pixel[a+1], pixel[b+1] = pixel[b+1], pixel[a+1]
	pixel[a+2], pixel[b+2] = pixel[b+2], pixel[a+2]
}

func shellSortSandwitch(keys []uint32, pixel []byte, data *Header) ([]uint32, []byte) {
	count := 0
	n := len(keys)
	if n == 2 {
		return nil, nil
	}
	for gi := len(sandwitchGaps) - 1; gi >= 0; gi-- {
		gap := sandwitchGaps[gi]
		if gap > n {
			continue
		}
		for i := gap; i < n; i++ {
			j := i
			for j >= gap && keys[j-gap] > keys[j] {
				count++
				swap(keys, pixel, j-gap, j)
				j -= gap
				if count%25000 == 0 {
					name := fmt.Sprintf("videos/frame_%05d.bmp", count/25000)
					WriteBMP(name, data.width, data.height, pixel)
				}
			}
		}
	}
	return keys, pixel
}

func WriteBMP(path string, width, height int, pixels []byte) error {
	stride := ((width*3 + 3) / 4) * 4
	pixelBytes := stride * height
	fileSize := 54 + pixelBytes
	out := make([]byte, fileSize)

	binary.LittleEndian.PutUint16(out[0:2], 0x4D42)           // BM
	binary.LittleEndian.PutUint32(out[2:6], uint32(fileSize)) // Размер файла
	binary.LittleEndian.PutUint16(out[6:8], 0)                // Резерв 1
	binary.LittleEndian.PutUint16(out[8:10], 0)               // Резерв 2
	binary.LittleEndian.PutUint32(out[10:14], 54)             // Смещение

	binary.LittleEndian.PutUint32(out[14:18], 40)                 // Размер заголовка
	binary.LittleEndian.PutUint32(out[18:22], uint32(width))      // Ширина
	binary.LittleEndian.PutUint32(out[22:26], uint32(height))     // Высота
	binary.LittleEndian.PutUint16(out[26:28], 1)                  // Число плоскостей (всегда = 1)
	binary.LittleEndian.PutUint16(out[28:30], 24)                 // Глубина цвета
	binary.LittleEndian.PutUint32(out[30:34], 0)                  // Метод сжатия
	binary.LittleEndian.PutUint32(out[34:38], uint32(pixelBytes)) // Размер изображения

	for y := 0; y < height; y++ {
		srcStart := y * width * 3
		dstStart := 54 + y*stride
		copy(out[dstStart:dstStart+width*3], pixels[srcStart:srcStart+width*3])
	}
	return os.WriteFile(path, out, 0644)

}

func MainFunc(path_to_bmp string, path_to_txt string, path_to_output string) {
	err, keys := ReadKeys(path_to_txt)
	if err != nil {
		log.Fatal(err)
	}
	structure, data, err := ReadBMP(path_to_bmp)
	if err != nil {
		log.Fatal(err)
	}
	pixels := ReadBGR(structure, data)
	keys, pixels = shellSortSandwitch(keys, pixels, structure)
	err = WriteBMP(path_to_output, structure.width, structure.height, pixels)
	if err != nil {
		log.Fatal(err)
	}
}
