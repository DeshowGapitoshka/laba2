package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type State struct {
	W, H   int
	pixels []byte
}

type Game struct {
	state *State
	delay *int64
	img   *ebiten.Image
}

type Header struct {
	smesh      uint32
	width      int
	height     int
	colorDepth uint16
}

func (g *Game) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		*g.delay = *g.delay + 100_000_000
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		*g.delay = *g.delay - 1_000_000
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	w, h, bgr := g.state.W, g.state.H, g.state.pixels
	rgba := make([]byte, 4*w*h) // Формат пикселей alpha-premultiplied RGBA
	for i := 0; i < w*h; i++ {
		rgba[i*4] = bgr[i*3+2]
		rgba[i*4+1] = bgr[i*3+1]
		rgba[i*4+2] = bgr[i*3]
		rgba[i*4+3] = 255
	}
	g.img.WritePixels(rgba)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1, -1)                     // Отразить по вертикали
	op.GeoM.Translate(0, float64(g.state.H)) // Сдвиг вниз
	screen.DrawImage(g.img, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

var sandwitchGaps = []int{
	1, 5, 19, 41, 109, 209, 505, 929, 2161, 3905,
	8929, 16001, 36289, 64769, 146305, 260609,
	587521, 1045505, 2354689, 4196353,
}

func ReadBMP(path string, g *Game) (*Header, []byte, error) {
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
	g.state.W, g.state.H = int(width), int(height)
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

func ReadKeys(path string, structure *Header, pixels []byte) ([]uint32, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("Ошибка чтения файла")
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("Пустой файл ключей")
	}

	tok := strings.Fields(string(data))
	count, err := strconv.Atoi(tok[0])
	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("Невалидный формат данных в документе с ключами")
	}
	if count != len(tok[1:]) {
		return nil, fmt.Errorf("Ожидалось %d ключей, получено %d", count, len(tok[1:]))
	}
	keys := make([]uint32, len(tok[1:]))
	for i := 0; i < len(tok[1:]); i++ {
		temp, err := strconv.Atoi(tok[1+i])
		if err != nil {
			log.Println(err)
			return nil, fmt.Errorf("Ошибка перевода строки в число")
		}
		keys[i] = uint32(temp)
	}
	return keys, nil
}

func swap(keys []uint32, pixel []byte, i, j int) {
	keys[i], keys[j] = keys[j], keys[i]
	a, b := i*3, j*3
	pixel[a], pixel[b] = pixel[b], pixel[a]
	pixel[a+1], pixel[b+1] = pixel[b+1], pixel[a+1]
	pixel[a+2], pixel[b+2] = pixel[b+2], pixel[a+2]
}

func shellSortSandwitch(keys []uint32, pixel []byte, g *Game, delay *int64) {
	count := 0
	n := len(keys)
	for gi := len(sandwitchGaps) - 1; gi >= 0; gi-- {
		gap := sandwitchGaps[gi]
		if gap > n {
			continue
		}
		for i := gap; i < n; i++ {
			j := i
			for j >= gap && keys[j-gap] > keys[j] {
				swap(keys, pixel, j-gap, j)
				count++
				j -= gap
				if count%10000 == 0 {
					if d := atomic.LoadInt64(delay); d > 0 {
						log.Println(*delay)
						time.Sleep(time.Duration(d))
					}
				}
			}
		}
	}
}

func main() {
	d := int64(1_000_000_000)
	game := &Game{state: &State{}, delay: &d}

	structure, data, err := ReadBMP("shuffled.bmp", game)
	if err != nil {
		log.Fatal(err)
	}
	game.img = ebiten.NewImage(structure.width, structure.height)
	pixels := ReadBGR(structure, data)
	keys, err2 := ReadKeys("keys.txt", structure, pixels)
	if err2 != nil {
		log.Fatal(err2)
	}

	game.state.W = structure.width
	game.state.H = structure.height
	game.state.pixels = pixels

	go func() {
		shellSortSandwitch(keys, pixels, game, game.delay)
	}()

	ebiten.SetWindowSize(1080, 814)
	ebiten.SetWindowTitle("Sorting")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}

}
