ffmpeg -framerate 25 -i videos/frame_%05d.bmp -c:v libx264 -pix_fmt yuv420p sort.mp4
go run main.go restore --bmp shuffled.bmp --keys keys.txt --out restored.bmp --video // Видео
go run main.go restore --bmp shuffled.bmp --keys keys.txt --out restored.bmp // Один прогон
go run main.go restore --bmp shuffled.bmp --keys keys.txt --out restored.bmp --bench // Несколько прогонов