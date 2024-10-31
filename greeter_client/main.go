package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os" // Импорт os для работы с переменными среды
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

const (
	defaultName = "world"
)

var (
	addr         = flag.String("addr", "host.docker.internal:50051", "the gRPC server address to connect to")
	name         = flag.String("name", defaultName, "Name to greet")
	influxURL    = "http://host.docker.internal:8086"
	influxToken  = "Rn9N-JY8vyj74zhfGXBub8E66E3fRV-cYZjMoBCilMURGp6aiY4yxMYggYQ0tZXnkpGxuWtNnJXR8vtBE5SKKg=="
	influxOrg    = "test"
	influxBucket = "bucket01"
)

func main() {
	flag.Parse()

	// Получаем значение порта из переменной среды с дефолтным значением 8090
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	// Инициализация клиента InfluxDB
	influxClient := influxdb2.NewClient(influxURL, influxToken)
	defer influxClient.Close()
	writeAPI := influxClient.WriteAPIBlocking(influxOrg, influxBucket)

	http.HandleFunc("/greet", func(w http.ResponseWriter, r *http.Request) {
		// Подключение к gRPC-серверу
		conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			http.Error(w, "Failed to connect to gRPC server", http.StatusInternalServerError)
			log.Fatalf("did not connect: %v", err)
			return
		}
		defer conn.Close()
		c := pb.NewGreeterClient(conn)

		// Получаем значение "name" из HTTP-запроса
		nameParam := r.URL.Query().Get("name")
		if nameParam == "" {
			nameParam = *name
		}

		// Измеряем время выполнения запроса
		startTime := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		response, err := c.SayHello(ctx, &pb.HelloRequest{Name: nameParam})
		elapsedTime := time.Since(startTime)

		// Сбор метрик
		status := "success"
		if err != nil {
			status = "error"
			http.Error(w, "Error executing gRPC request", http.StatusInternalServerError)
			log.Printf("could not greet: %v", err)
		} else {
			fmt.Fprintf(w, "Greeting: %s", response.GetMessage())
			log.Printf("Greeting: %s", response.GetMessage())
		}
		// Запись метрики в InfluxDB асинхронно
		p := influxdb2.NewPointWithMeasurement("grpc_requests").
			AddTag("status", status).
			AddField("name", nameParam).
			AddField("elapsed_time", elapsedTime.Milliseconds()).
			SetTime(startTime)

		go func() {
			if err := writeAPI.WritePoint(context.Background(), p); err != nil {
				log.Printf("Error writing to InfluxDB: %v", err)
			}
		}()
	})

	log.Printf("HTTP сервер запущен на порту %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
