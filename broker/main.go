package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// --- Structs ---

type DadosSensor struct {
	ID    string  `json:"id"`
	Tipo  string  `json:"tipo"`
	Valor float64 `json:"valor"`
}

type PontoHistorico struct {
	V float64 `json:"v"`
	T int64   `json:"t"`
}

type Agregador struct {
	Soma  float64
	Conta int
}

type Acao struct {
	Hora      string `json:"hora"`
	Descricao string `json:"descricao"`
	Origem    string `json:"origem"`
	Status    string `json:"status"`
}

type Alerta struct {
	Hora     string `json:"hora"`
	Mensagem string `json:"mensagem"`
	Nivel    string `json:"nivel"`
}

type StatusSensor struct {
	UltimoSinal time.Time
	Online      bool
}

type StatusAtuador struct {
	Nome        string
	Porta       string
	Online      bool
	UltimoTeste time.Time
}

type InfoSensor struct {
	Online      bool   `json:"online"`
	UltimoSinal string `json:"ultimo_sinal"`
}

type InfoAtuador struct {
	Online bool   `json:"online"`
	Nome   string `json:"nome"`
}

type StatusResposta struct {
	Sensores        map[string][]PontoHistorico `json:"sensores"`
	Acoes           []Acao                      `json:"acoes"`
	Alertas         []Alerta                    `json:"alertas"`
	Estado          map[string]bool             `json:"estado"`
	StatusSensores  map[string]InfoSensor       `json:"status_sensores"`
	StatusAtuadores map[string]InfoAtuador      `json:"status_atuadores"`
}

type EstadoSalvo struct {
	Sensores           map[string][]PontoHistorico `json:"sensores"`
	Acoes              []Acao                      `json:"acoes"`
	Alertas            []Alerta                    `json:"alertas"`
	ChillerMostoLigado bool                        `json:"chiller_mosto"`
	ChillerAdegaLigado bool                        `json:"chiller_adega"`
	BombaLigada        bool                        `json:"bomba"`
	EfeitoTempMosto    float64                     `json:"efeito_temp_mosto"`
	EfeitoTempAdega    float64                     `json:"efeito_temp_adega"`
	EfeitoDensidade    float64                     `json:"efeito_densidade"`
	EfeitoNivel        float64                     `json:"efeito_nivel"`
	AguardandoNovaLeva bool                        `json:"aguardando_nova_leva"`
	UltimoValor        map[string]float64          `json:"ultimo_valor"`
}

// --- Constantes ---
const (
	RetencaoMinutos  = 10
	MaxAcoes         = 5
	MaxAlertas       = 10
	ArquivoJSON      = "estado_broker.json"
	TimeoutSensor    = 10 * time.Second
	IntervaloAtuador = 15 * time.Second
)

// --- Variáveis globais ---
var (
	estadoSensores = make(map[string][]PontoHistorico)
	bufferTemp     = make(map[string]*Agregador)
	ultimoValor    = make(map[string]float64)
	mu             sync.RWMutex

	chillerMostoLigado bool
	chillerAdegaLigado bool
	bombaLigada        bool

	efeitoTempMosto float64 = 0
	efeitoTempAdega float64 = 0
	efeitoDensidade float64 = 0
	efeitoNivel     float64 = 0

	aguardandoNovaLeva bool    = false
	densidadeNovaLeva  float64 = 0

	bombaCooldown    bool = false
	bombaCooldownFim time.Time

	historicoAcoes   []Acao
	historicoAlertas []Alerta

	atuadorHost  string
	ultimoAlerta = make(map[string]time.Time)

	statusSensores = make(map[string]*StatusSensor)
	atuadores      = []StatusAtuador{
		{Nome: "Refrig. Mosto", Porta: ":9090", Online: false},
		{Nome: "Bomba Trasfega", Porta: ":9091", Online: false},
		{Nome: "Refrig. Adega", Porta: ":9092", Online: false},
	}
)

func main() {
	atuadorHost = os.Getenv("ATUADOR_HOST")
	if atuadorHost == "" {
		atuadorHost = "172.16.103.13"
	}

	fmt.Println("=========================================================")
	fmt.Println("🚀 INICIANDO BROKER...")

	carregarDados()

	go escutarSensoresUDP()
	go processarAgrupamento()
	go escutarClienteTCP()
	go monitorarAtuadores()
	go rotinaDeSalvamento()

	fmt.Println("📡 Escutando Sensores em UDP :8080")
	fmt.Println("💻 Escutando Cliente Web em TCP :8081")
	fmt.Printf("⚙️  Enviando comandos para Atuadores em: %s\n", atuadorHost)
	fmt.Println("=========================================================\n")

	select {}
}

// --- Funções de Persistência (Salvar e Carregar) ---
func carregarDados() {
	mu.Lock()
	defer mu.Unlock()

	file, err := os.ReadFile(ArquivoJSON)
	if err == nil {
		var salvo EstadoSalvo
		if err := json.Unmarshal(file, &salvo); err == nil {
			if salvo.Sensores != nil {
				estadoSensores = salvo.Sensores
			}
			if salvo.Acoes != nil {
				historicoAcoes = salvo.Acoes
			}
			if salvo.Alertas != nil {
				historicoAlertas = salvo.Alertas
			}
			if salvo.UltimoValor != nil {
				ultimoValor = salvo.UltimoValor
			}

			chillerMostoLigado = salvo.ChillerMostoLigado
			chillerAdegaLigado = salvo.ChillerAdegaLigado
			bombaLigada = salvo.BombaLigada
			efeitoTempMosto = salvo.EfeitoTempMosto
			efeitoTempAdega = salvo.EfeitoTempAdega
			efeitoDensidade = salvo.EfeitoDensidade
			efeitoNivel = salvo.EfeitoNivel
			aguardandoNovaLeva = salvo.AguardandoNovaLeva

			fmt.Println("💾 [OK] Estado anterior carregado com sucesso!")
			return
		}
	}
	fmt.Println("💾 [INFO] Nenhum estado anterior encontrado. Iniciando do zero.")
}

func rotinaDeSalvamento() {
	for {
		time.Sleep(5 * time.Second)

		mu.RLock()
		salvo := EstadoSalvo{
			Sensores:           estadoSensores,
			Acoes:              historicoAcoes,
			Alertas:            historicoAlertas,
			ChillerMostoLigado: chillerMostoLigado,
			ChillerAdegaLigado: chillerAdegaLigado,
			BombaLigada:        bombaLigada,
			EfeitoTempMosto:    efeitoTempMosto,
			EfeitoTempAdega:    efeitoTempAdega,
			EfeitoDensidade:    efeitoDensidade,
			EfeitoNivel:        efeitoNivel,
			AguardandoNovaLeva: aguardandoNovaLeva,
			UltimoValor:        ultimoValor,
		}
		dados, err := json.MarshalIndent(salvo, "", "  ")
		mu.RUnlock()

		if err == nil {
			tempFile := ArquivoJSON + ".tmp"
			os.WriteFile(tempFile, dados, 0644)
			os.Rename(tempFile, ArquivoJSON)
		}
	}
}

// --- Monitoramento de Rede ---
func verificarSensoresOffline() {
	mu.Lock()
	defer mu.Unlock()

	agora := time.Now()
	for tipo, st := range statusSensores {
		estaOnline := agora.Sub(st.UltimoSinal) < TimeoutSensor
		if st.Online && !estaOnline {
			st.Online = false
			hora := st.UltimoSinal.Format("15:04:05")
			go adicionarAlerta(fmt.Sprintf("Sensor '%s' fora de ar! Último dado: %s", tipo, hora), "CRITICO")
			fmt.Printf("\n[❌ ALERTA] Sensor '%s' parou de responder.\n", tipo)
		}
	}
}

func monitorarAtuadores() {
	for {
		time.Sleep(IntervaloAtuador)

		mu.Lock()
		for i := range atuadores {
			at := &atuadores[i]
			conn, err := net.DialTimeout("tcp", atuadorHost+at.Porta, 2*time.Second)
			eraOnline := at.Online
			if err == nil {
				conn.Close()
				at.Online = true
				at.UltimoTeste = time.Now()
				if !eraOnline {
					go adicionarAlerta(fmt.Sprintf("Atuador '%s' voltou!", at.Nome), "AVISO")
					fmt.Printf("\n[✅ INFO] Atuador '%s' voltou online.\n", at.Nome)
				}
			} else {
				at.Online = false
				if eraOnline {
					go adicionarAlerta(fmt.Sprintf("Atuador '%s' fora de ar!", at.Nome), "CRITICO")
					fmt.Printf("\n[❌ ALERTA] Atuador '%s' ficou offline.\n", at.Nome)
				}
			}
		}
		mu.Unlock()
	}
}

// --- UDP: recebe dados dos sensores ---
func escutarSensoresUDP() {
	addr, _ := net.ResolveUDPAddr("udp", ":8080")
	conn, _ := net.ListenUDP("udp", addr)
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		var sensor DadosSensor
		if json.Unmarshal(buf[:n], &sensor) == nil {
			fmt.Printf("[📡 SENSOR] %s: %.3f\n", sensor.Tipo, sensor.Valor)

			mu.Lock()
			if bufferTemp[sensor.Tipo] == nil {
				bufferTemp[sensor.Tipo] = &Agregador{}
			}
			bufferTemp[sensor.Tipo].Soma += sensor.Valor
			bufferTemp[sensor.Tipo].Conta++

			agora := time.Now()
			if statusSensores[sensor.Tipo] == nil {
				statusSensores[sensor.Tipo] = &StatusSensor{}
			}
			eraOffline := !statusSensores[sensor.Tipo].Online
			statusSensores[sensor.Tipo].UltimoSinal = agora
			statusSensores[sensor.Tipo].Online = true
			mu.Unlock()

			if eraOffline {
				go adicionarAlerta(fmt.Sprintf("Sensor '%s' voltou!", sensor.Tipo), "AVISO")
			}
		}
	}
}

// --- Motor principal: agrega, aplica física, checa automação ---
func processarAgrupamento() {
	for {
		time.Sleep(2 * time.Second)

		verificarSensoresOffline()

		mu.Lock()

		// --- Física do Sistema ---
		if chillerMostoLigado {
			efeitoTempMosto -= 0.4 * (1.0 - math.Abs(efeitoTempMosto)/10.0)
			if efeitoTempMosto < -9.0 {
				efeitoTempMosto = -9.0
			}
		} else {
			efeitoTempMosto += 0.15
			if efeitoTempMosto > 0 {
				efeitoTempMosto = 0
			}
		}

		if chillerAdegaLigado {
			efeitoTempAdega -= 0.25 * (1.0 - math.Abs(efeitoTempAdega)/5.0)
			if efeitoTempAdega < -4.0 {
				efeitoTempAdega = -4.0
			}
		} else {
			efeitoTempAdega += 0.1
			if efeitoTempAdega > 0 {
				efeitoTempAdega = 0
			}
		}

		tempMostoAtual := ultimoValor["Temperatura do Mosto"]
		if tempMostoAtual == 0 {
			tempMostoAtual = 22.0
		}

		if aguardandoNovaLeva {
			nivelAtual := ultimoValor["Nível da Dorna"]
			if nivelAtual > 60.0 {
				aguardandoNovaLeva = false
				densidadeNovaLeva = 1.050 + (float64(time.Now().UnixNano()%300))/10000.0
				efeitoDensidade = densidadeNovaLeva - 1.060
				go adicionarAlerta(fmt.Sprintf("🍇 Nova leva carregada (dens: %.3f)", densidadeNovaLeva), "AVISO")
			}
		} else {
			taxaFermentacao := 0.0003
			if tempMostoAtual > 24.0 {
				taxaFermentacao = 0.0008
			} else if tempMostoAtual > 22.0 {
				taxaFermentacao = 0.0005
			} else if tempMostoAtual < 16.0 {
				taxaFermentacao = 0.0001
			}
			efeitoDensidade -= taxaFermentacao
		}

		if efeitoDensidade < -0.088 {
			efeitoDensidade = -0.088
		}

		if bombaLigada {
			efeitoNivel -= 1.5
			if efeitoNivel < -90.0 {
				efeitoNivel = -90.0
			}
		} else if aguardandoNovaLeva {
			efeitoNivel += 0.8
			if efeitoNivel > 0 {
				efeitoNivel = 0
			}
		} else {
			efeitoNivel += 0.02
			if efeitoNivel > 0 {
				efeitoNivel = 0
			}
		}

		now := time.Now().Unix()
		cutoff := time.Now().Add(-RetencaoMinutos * time.Minute).Unix()

		// --- Agrupamento e Limitação de Casas Decimais ---
		for tipo, ag := range bufferTemp {
			if ag.Conta == 0 {
				continue
			}
			if st, ok := statusSensores[tipo]; !ok || !st.Online {
				continue
			}

			media := ag.Soma / float64(ag.Conta)
			ag.Soma = 0
			ag.Conta = 0

			switch tipo {
			case "Temperatura do Mosto":
				media += efeitoTempMosto
				media = clamp(media, 12.0, 35.0)
			case "Temp da Adega":
				media += efeitoTempAdega
				media = clamp(media, 8.0, 22.0)
			case "Densidade do Mosto":
				media += efeitoDensidade
				media = clamp(media, 0.990, 1.095)
			case "Nível da Dorna":
				media += efeitoNivel
				media = clamp(media, 0.0, 100.0)
			}

			// LIMITADOR: Arredonda para 1 casa decimal (exceto densidade)
			if tipo == "Densidade do Mosto" {
				ultimoValor[tipo] = math.Round(media*1000) / 1000 // 3 casas: 1.055
			} else {
				ultimoValor[tipo] = math.Round(media*10) / 10 // 1 casa: 22.5
			}

			estadoSensores[tipo] = append(estadoSensores[tipo], PontoHistorico{
				V: ultimoValor[tipo],
				T: now,
			})

			// Limpeza de histórico antigo
			i := 0
			for i < len(estadoSensores[tipo]) && estadoSensores[tipo][i].T < cutoff {
				i++
			}
			if i > 0 {
				estadoSensores[tipo] = estadoSensores[tipo][i:]
			}
		}

		mu.Unlock()

		go verificarAutomacao()
		go verificarAlertas()
	}
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

// --- Automação e Fail-Safe ---
func verificarAutomacao() {
	mu.RLock()
	tempMosto := ultimoValor["Temperatura do Mosto"]
	tempAdega := ultimoValor["Temp da Adega"]
	densidade := ultimoValor["Densidade do Mosto"]
	nivel := ultimoValor["Nível da Dorna"]

	cMosto := chillerMostoLigado
	cAdega := chillerAdegaLigado
	bomba := bombaLigada
	esperandoLeva := aguardandoNovaLeva
	emCooldown := bombaCooldown && time.Now().Before(bombaCooldownFim)

	// Lendo status online dos sensores
	mostoOnline := statusSensores["Temperatura do Mosto"] != nil && statusSensores["Temperatura do Mosto"].Online
	adegaOnline := statusSensores["Temp da Adega"] != nil && statusSensores["Temp da Adega"].Online
	densidadeOnline := statusSensores["Densidade do Mosto"] != nil && statusSensores["Densidade do Mosto"].Online
	nivelOnline := statusSensores["Nível da Dorna"] != nil && statusSensores["Nível da Dorna"].Online
	mu.RUnlock()

	// ==========================================
	// 1. SISTEMA DE PROTEÇÃO (FAIL-SAFE)
	// Se o sensor cair enquanto o atuador está ligado, desliga imediatamente!
	// ==========================================

	if cMosto && !mostoOnline {
		mu.Lock()
		chillerMostoLigado = false
		mu.Unlock()
		go registrarEEnviarComando("Emergência: Desligar Refrig. Mosto", ":9090", "DESLIGAR_REFRIG_MOSTO", "AUTO")
		go adicionarAlerta("Sensor do Mosto caiu! Refrigeração desligada por segurança.", "CRITICO")
	}

	if cAdega && !adegaOnline {
		mu.Lock()
		chillerAdegaLigado = false
		mu.Unlock()
		go registrarEEnviarComando("Emergência: Desligar Refrig. Adega", ":9092", "DESLIGAR_REFRIG_ADEGA", "AUTO")
		go adicionarAlerta("Sensor da Adega caiu! Refrigeração desligada por segurança.", "CRITICO")
	}

	if bomba && (!nivelOnline || !densidadeOnline) {
		mu.Lock()
		bombaLigada = false
		aguardandoNovaLeva = true
		bombaCooldown = true
		bombaCooldownFim = time.Now().Add(30 * time.Second)
		mu.Unlock()
		go registrarEEnviarComando("Emergência: Parar Bomba", ":9091", "PARAR_BOMBA", "AUTO")
		go adicionarAlerta("Sensores caíram! Bomba parada por segurança.", "CRITICO")
	}

	// ==========================================
	// 2. LÓGICA NORMAL DE AUTOMAÇÃO
	// Só avalia se os sensores responsáveis estiverem online
	// ==========================================

	if mostoOnline {
		if tempMosto > 24.5 && !cMosto {
			mu.Lock()
			chillerMostoLigado = true
			mu.Unlock()
			go registrarEEnviarComando("Ligar Refrig. Mosto", ":9090", "LIGAR_REFRIG_MOSTO", "AUTO")
		} else if tempMosto < 17.5 && cMosto {
			mu.Lock()
			chillerMostoLigado = false
			mu.Unlock()
			go registrarEEnviarComando("Desligar Refrig. Mosto", ":9090", "DESLIGAR_REFRIG_MOSTO", "AUTO")
		}
	}

	if adegaOnline {
		if tempAdega > 16.5 && !cAdega {
			mu.Lock()
			chillerAdegaLigado = true
			mu.Unlock()
			go registrarEEnviarComando("Ligar Refrig. Adega", ":9092", "LIGAR_REFRIG_ADEGA", "AUTO")
		} else if tempAdega < 11.5 && cAdega {
			mu.Lock()
			chillerAdegaLigado = false
			mu.Unlock()
			go registrarEEnviarComando("Desligar Refrig. Adega", ":9092", "DESLIGAR_REFRIG_ADEGA", "AUTO")
		}
	}

	if densidadeOnline && nivelOnline {
		if densidade > 0 && densidade <= 1.010 && !bomba && !esperandoLeva && !emCooldown {
			mu.Lock()
			bombaLigada = true
			mu.Unlock()
			go registrarEEnviarComando("Acionar Bomba - Fermentação concluída", ":9091", "ACIONAR_BOMBA", "AUTO")
			go adicionarAlerta("Fermentação concluída — trasfega iniciada automaticamente", "AVISO")
		}

		if nivel > 0 && nivel <= 5.0 && bomba {
			mu.Lock()
			bombaLigada = false
			aguardandoNovaLeva = true
			bombaCooldown = true
			bombaCooldownFim = time.Now().Add(30 * time.Second)
			mu.Unlock()
			go registrarEEnviarComando("Parar Bomba - Dorna vazia", ":9091", "PARAR_BOMBA", "AUTO")
			go adicionarAlerta("Dorna vazia — bomba desligada. Aguardando nova leva...", "AVISO")
		}
	}
}

func verificarAlertas() {
	mu.RLock()
	tempMosto := ultimoValor["Temperatura do Mosto"]
	tempAdega := ultimoValor["Temp da Adega"]
	mu.RUnlock()

	if tempMosto > 27.0 {
		adicionarAlerta(fmt.Sprintf("Temperatura do Mosto crítica: %.1f°C!", tempMosto), "CRITICO")
	} else if tempMosto > 0 && tempMosto < 15.0 {
		adicionarAlerta(fmt.Sprintf("Temperatura do Mosto muito baixa: %.1f°C!", tempMosto), "CRITICO")
	}
	if tempAdega > 18.0 {
		adicionarAlerta(fmt.Sprintf("Temperatura da Adega elevada: %.1f°C", tempAdega), "AVISO")
	}
}

func adicionarAlerta(mensagem, nivel string) {
	mu.Lock()
	defer mu.Unlock()

	if t, ok := ultimoAlerta[mensagem]; ok && time.Since(t) < 5*time.Minute {
		return
	}
	ultimoAlerta[mensagem] = time.Now()

	if nivel == "CRITICO" {
		fmt.Printf("\n[🚨 ALERTA CRÍTICO] %s\n\n", mensagem)
	} else {
		fmt.Printf("\n[⚠️ AVISO] %s\n\n", mensagem)
	}

	alerta := Alerta{Hora: time.Now().Format("15:04:05"), Mensagem: mensagem, Nivel: nivel}
	historicoAlertas = append([]Alerta{alerta}, historicoAlertas...)

	if len(historicoAlertas) > MaxAlertas {
		historicoAlertas = historicoAlertas[:MaxAlertas]
	}
}

// --- Envia comando ---
func registrarEEnviarComando(nomeAcao, porta, comando, origem string) {
	mu.RLock()
	atuadorOnline := false
	nomeAtuador := porta
	for _, at := range atuadores {
		if at.Porta == porta {
			atuadorOnline = at.Online
			nomeAtuador = at.Nome
			break
		}
	}
	mu.RUnlock()

	if !atuadorOnline {
		msg := fmt.Sprintf("Comando descartado — atuador '%s' fora de ar", nomeAtuador)
		fmt.Printf("\n[⚠️ AVISO] %s\n\n", msg)
		go adicionarAlerta(msg, "CRITICO")

		acao := Acao{
			Hora:      time.Now().Format("15:04:05"),
			Descricao: nomeAcao + " (DESCARTADO)",
			Origem:    origem,
			Status:    "FALHA",
		}
		mu.Lock()
		historicoAcoes = append([]Acao{acao}, historicoAcoes...)
		if len(historicoAcoes) > MaxAcoes {
			historicoAcoes = historicoAcoes[:MaxAcoes]
		}
		mu.Unlock()
		return
	}

	status := "FALHA"
	conn, err := net.DialTimeout("tcp", atuadorHost+porta, 2*time.Second)
	if err == nil {
		fmt.Fprintf(conn, "%s\n", comando)
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		resp, readErr := bufio.NewReader(conn).ReadString('\n')
		conn.Close()
		if readErr == nil && strings.Contains(resp, "CONFIRMADO") {
			status = "CONFIRMADO"
		}
	} else {
		mu.Lock()
		for i := range atuadores {
			if atuadores[i].Porta == porta {
				atuadores[i].Online = false
			}
		}
		mu.Unlock()
		go adicionarAlerta(fmt.Sprintf("Atuador '%s' fora de ar!", nomeAtuador), "CRITICO")
	}

	fmt.Printf("\n[⚙️ AÇÃO - %s] %s -> Status: %s\n\n", origem, nomeAcao, status)

	acao := Acao{
		Hora:      time.Now().Format("15:04:05"),
		Descricao: nomeAcao,
		Origem:    origem,
		Status:    status,
	}
	mu.Lock()
	historicoAcoes = append([]Acao{acao}, historicoAcoes...)
	if len(historicoAcoes) > MaxAcoes {
		historicoAcoes = historicoAcoes[:MaxAcoes]
	}
	mu.Unlock()
}

// --- TCP: atende o cliente web ---
func escutarClienteTCP() {
	ln, _ := net.Listen("tcp", ":8081")
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err == nil {
			go tratarRequisicaoCliente(conn)
		}
	}
}

func tratarRequisicaoCliente(conn net.Conn) {
	defer conn.Close()
	mensagem, _ := bufio.NewReader(conn).ReadString('\n')
	req := strings.TrimSpace(mensagem)

	switch {
	case req == "GET_STATUS":
		mu.RLock()

		infoSensores := make(map[string]InfoSensor)
		for tipo, st := range statusSensores {
			infoSensores[tipo] = InfoSensor{
				Online:      st.Online,
				UltimoSinal: st.UltimoSinal.Format("15:04:05"),
			}
		}

		infoAtuadores := make(map[string]InfoAtuador)
		for _, at := range atuadores {
			infoAtuadores[at.Porta] = InfoAtuador{
				Online: at.Online,
				Nome:   at.Nome,
			}
		}

		resp := StatusResposta{
			Sensores: estadoSensores,
			Acoes:    historicoAcoes,
			Alertas:  historicoAlertas,
			Estado: map[string]bool{
				"chiller_mosto": chillerMostoLigado,
				"chiller_adega": chillerAdegaLigado,
				"bomba":         bombaLigada,
			},
			StatusSensores:  infoSensores,
			StatusAtuadores: infoAtuadores,
		}
		mu.RUnlock()
		dados, _ := json.Marshal(resp)
		fmt.Fprintf(conn, "%s\n", string(dados))

	case strings.HasPrefix(req, "CMD:"):
		partes := strings.Split(req, ":")
		if len(partes) == 3 {
			porta := ":" + partes[1]
			comando := partes[2]
			nome := traduzirComando(comando)

			mu.Lock()
			aplicarComando(comando)
			mu.Unlock()

			go registrarEEnviarComando(nome, porta, comando, "MANUAL")
			fmt.Fprintf(conn, "OK\n")
		}
	}
}

func aplicarComando(cmd string) {
	switch cmd {
	case "LIGAR_REFRIG_MOSTO":
		chillerMostoLigado = true
	case "DESLIGAR_REFRIG_MOSTO":
		chillerMostoLigado = false
	case "LIGAR_REFRIG_ADEGA":
		chillerAdegaLigado = true
	case "DESLIGAR_REFRIG_ADEGA":
		chillerAdegaLigado = false
	case "ACIONAR_BOMBA":
		bombaLigada = true
	case "PARAR_BOMBA":
		bombaLigada = false
	}
}

func traduzirComando(cmd string) string {
	nomes := map[string]string{
		"LIGAR_REFRIG_MOSTO":    "Ligar Refrig. Mosto",
		"DESLIGAR_REFRIG_MOSTO": "Desligar Refrig. Mosto",
		"LIGAR_REFRIG_ADEGA":    "Ligar Refrig. Adega",
		"DESLIGAR_REFRIG_ADEGA": "Desligar Refrig. Adega",
		"ACIONAR_BOMBA":         "Acionar Bomba Trasfega",
		"PARAR_BOMBA":           "Parar Bomba",
	}
	if n, ok := nomes[cmd]; ok {
		return n
	}
	return cmd
}
