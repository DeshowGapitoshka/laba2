package main

import (
	"encoding/binary"
	"fmt"
	video "lab2/video"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

var sandwitchGaps = []int{
	1, 5, 19, 41, 109, 209, 505, 929, 2161, 3905,
	8929, 16001, 36289, 64769, 146305, 260609,
	587521, 1045505, 2354689, 4196353,
}

type Header struct {
	smesh      uint32
	width      int
	height     int
	colorDepth uint16
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

func ReadKeys(path string, structure *Header, pixels []byte) (error, []uint32) {
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
	if count != (len(pixels)-int(structure.smesh))/3 {
		return fmt.Errorf("Количество ключей не совпадает с длиной пикселей"), nil
	}
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

func shellSortSandwitch(keys []uint32, pixel []byte) ([]uint32, []byte, time.Duration) {
	start := time.Now()
	n := len(keys)
	if n == 2 {
		return nil, nil, 0
	}
	for gi := len(sandwitchGaps) - 1; gi >= 0; gi-- {
		gap := sandwitchGaps[gi]
		if gap > n {
			continue
		}
		for i := gap; i < n; i++ {
			j := i
			for j >= gap && keys[j-gap] > keys[j] {
				swap(keys, pixel, j-gap, j)
				j -= gap
			}
		}
	}
	time := time.Since(start)
	return keys, pixel, time
}

func shellSortHibard(keys []uint32, pixel []byte) ([]uint32, []byte, time.Duration) {
	start := time.Now()
	n := len(keys)
	g := 1
	i := 2
	var hibardGaps []int
	for g < n {
		hibardGaps = append(hibardGaps, g)
		g = int(math.Pow(2, float64(i))) - 1
		i++
	}
	for gi := len(hibardGaps) - 1; gi >= 0; gi-- {
		gap := hibardGaps[gi]
		if gap > n {
			continue
		}
		for i := gap; i < n; i++ {
			j := i
			for j >= gap && keys[j-gap] > keys[j] {
				swap(keys, pixel, j-gap, j)
				j -= gap
			}
		}
	}
	time := time.Since(start)
	return keys, pixel, time
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

func mediana(time []int) int {
	n := len(time)
	if n == 0 {
		return 0
	}
	sort.Ints(time)
	if n%2 == 1 {
		temp := time[n/2] + time[n/2+1]
		return temp
	} else {
		temp := time[n/2]
		return temp
	}
}

func mainFuncSandwitch(path_to_bmp string, path_to_txt string, path_to_output string, count int, flag bool) {
	timeSandwitch := make([]int, count)
	for i := 0; i < count; i++ {
		structure, data, err := ReadBMP(path_to_bmp)
		if err != nil {
			log.Fatal(err)
		}
		err, keys := ReadKeys(path_to_txt, structure, data)
		if err != nil {
			log.Fatal(err)
		}
		pixels := ReadBGR(structure, data)
		keys, pixels, temp_time_sandwitch := shellSortSandwitch(keys, pixels)
		err = WriteBMP(path_to_output, structure.width, structure.height, pixels)
		if err != nil {
			log.Fatal(err)
		}
		timeSandwitch = append(timeSandwitch, int(temp_time_sandwitch))
	}
	if flag == true {
		mediana_Sandwitch := mediana(timeSandwitch)
		fmt.Printf("Сортировка Шелла,Шаги Седжвика. Медиана:%.2f мс\n", float32(mediana_Sandwitch/1000000))
	}
}

func mainFuncHibard(path_to_bmp string, path_to_txt string, path_to_output string, count int, flag bool) {
	timeHibard := make([]int, count)
	for i := 0; i < count; i++ {
		structure, data, err := ReadBMP(path_to_bmp)
		if err != nil {
			log.Fatal(err)
		}
		err, keys := ReadKeys(path_to_txt, structure, data)
		if err != nil {
			log.Fatal(err)
		}
		pixels := ReadBGR(structure, data)
		keys, pixels, temp_time_hibard := shellSortHibard(keys, pixels)
		timeHibard = append(timeHibard, int(temp_time_hibard))
	}
	if flag == true {
		mediana_Hibard := mediana(timeHibard)
		fmt.Printf("Сортировка Шелла,Шаги Хиббарда. Медиана:%.2f мс\n", float32(mediana_Hibard/1000000))
	}
}
func progrev(path_to_bmp, path_to_txt, path_to_output string) {
	mainFuncSandwitch(path_to_bmp, path_to_txt, path_to_output, 3, false)
	mainFuncHibard(path_to_bmp, path_to_txt, path_to_output, 3, false)
}

func main() {
	args := os.Args
	if args[1] == "restore" && args[2] == "--bmp" && args[4] == "--keys" && args[6] == "--out" && len(args) == 8 {
		mainFuncSandwitch(args[3], args[5], args[7], 1, true)
		mainFuncHibard(args[3], args[5], args[7], 1, true)
		return
	} else if args[1] == "restore" && args[2] == "--bmp" && args[4] == "--keys" && args[6] == "--out" && len(args) == 10 {
		if args[8] == "--bench" {
			temp, err := strconv.Atoi(args[9])
			if err != nil {
				log.Println(err)
				os.Exit(1)
			}
			mainFuncSandwitch(args[3], args[5], args[7], temp, true)
			mainFuncHibard(args[3], args[5], args[7], temp, true)
			return
		}
	} else if args[1] == "restore" && args[2] == "--bmp" && args[4] == "--keys" && args[6] == "--out" && args[8] == "--video" && len(args) == 9 {
		video.MainFunc(args[3], args[5], args[7])
		return
	} else {
		log.Println("Неверные аргументы программы")
		os.Exit(1)
	}

}
