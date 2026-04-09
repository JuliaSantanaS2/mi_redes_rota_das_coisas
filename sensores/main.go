package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net"
	"os"
	"time"
)

// Estrutura da mensagem enviada p/ o servidor
type DadosSensor struct {
	ID    string  `json:"id"`
	Tipo  string  `json:"tipo"`
	Valor float64 `json:"valor"`
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

type simulador struct {
	base, valor, min, max, ruido, fase, amplitude float64
}

// Cria um novo simulador
func novoSimulador(base, min, max, ruido, amplitude float64) *simulador {
	return &simulador{
		base:      base,
		valor:     base,
		min:       min,
		max:       max,
		ruido:     ruido,
		amplitude: amplitude,
		fase:      rand.Float64() * math.Pi * 2,
	}
}

// Gera o próximo valor do sensor usando onda senoidal + ruído
func (s *simulador) proximo(t float64) float64 {
	onda := math.Sin(t*0.5+s.fase) * s.amplitude
	ruido := (rand.Float64()*2 - 1) * s.ruido

	s.valor = s.base + onda + ruido
	s.valor = clamp(s.valor, s.min, s.max)

	return s.valor
}

func main() {

	// Configuração do sensor padrão
	serverAddr := getEnv("SERVER_IP", "127.0.0.1:8080")
	sensorID := getEnv("SENSOR_ID", "SENS-01")
	sensorTipo := getEnv("SENSOR_TIPO", "Temperatura do Mosto")

	intervaloMs := 1

	// Configurações dos valores gerados dos sensores
	configs := map[string]struct {
		base, min, max, ruido, amplitude float64
	}{
		"Temperatura do Mosto": {22.0, 12.0, 35.0, 0.1, 0.8},
		"Densidade do Mosto":   {1.060, 0.990, 1.095, 0.001, 0.003},
		"Umidade da Adega":     {70.0, 40.0, 95.0, 0.3, 1.5},
		"Temp da Adega":        {14.0, 8.0, 22.0, 0.1, 0.5},
		"Nível da Dorna":       {80.0, 0.0, 100.0, 0.2, 0.5},
	}

	c, ok := configs[sensorTipo]
	if !ok {
		fmt.Println("❌ Tipo de sensor inválido:", sensorTipo)
		return
	}

	sim := novoSimulador(c.base, c.min, c.max, c.ruido, c.amplitude)

	brokerAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		fmt.Println("❌ Erro ao resolver endereço:", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, brokerAddr)
	if err != nil {
		fmt.Println("❌ Erro ao conectar no broker:", err)
		return
	}
	defer conn.Close()

	ticker := time.NewTicker(time.Duration(intervaloMs) * time.Millisecond)
	defer ticker.Stop()

	t := 0.0
	valorAtual := sim.proximo(t)
	contadorCiclos := 0

	fmt.Printf("🚀 Sensor %s (%s) enviando a cada 1ms | muda a cada 1000 envios\n",
		sensorID, sensorTipo)

	for range ticker.C {

		valorFormatado := math.Round(valorAtual*10) / 10

		// Dados do sensor para enviar
		dados := DadosSensor{
			ID:    sensorID,
			Tipo:  sensorTipo,
			Valor: valorFormatado,
		}

		payload, _ := json.Marshal(dados)
		_, _ = conn.Write(payload)

		if contadorCiclos%2 == 0 {
			fmt.Printf("📡 %s #%d -> %.1f\n",
				sensorTipo,
				contadorCiclos,
				dados.Valor)
		}

		contadorCiclos++

		// Quando o contador chegar em 1000, ele vai mudar o valor do sensor
		if contadorCiclos == 1000 {
			t += 1.0
			valorAtual = sim.proximo(t)
			contadorCiclos = 0
		}
	}
}
