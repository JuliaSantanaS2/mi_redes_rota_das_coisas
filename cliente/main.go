package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

const htmlPagina = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Vinicola - Painel de Controle</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        :root {
            --bg: #0c0a10;
            --surface: #16121f;
            --surface2: #1e1828;
            --border: #2d2540;
            --wine: #8B1A2E;
            --wine-glow: rgba(139,26,46,0.3);
            --gold: #d4a053;
            --text: #f0ecf7;
            --text-muted: #9b91af;
            --green: #2ecc71;
            --red: #e74c3c;
            --yellow: #f39c12;
            --blue: #3498db;
            --radius: 12px;
        }
        /* BANNER BROKER OFFLINE */
		#broker-offline {
		    display: none;
		    background: var(--red);
		    color: white;
		    text-align: center;
		    padding: 10px;
		    font-weight: bold;
		    border-radius: var(--radius);
		    margin-bottom: 20px;
		}
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: monospace;
            background: var(--bg);
            color: var(--text);
            min-height: 100vh;
            padding: 20px;
        }

        /* HEADER */
        .header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            background: var(--surface);
            border: 1px solid var(--border);
            border-radius: var(--radius);
            padding: 16px 24px;
            margin-bottom: 20px;
            box-shadow: 0 0 30px var(--wine-glow);
        }
        .header-left { display: flex; align-items: center; gap: 14px; }
        .header h1 { font-size: 1.3rem; font-weight: 600; color: var(--text); }
        .logo { font-size: 2rem; }
        .status-badges { display: flex; gap: 10px; flex-wrap: wrap; }
        .badge {
            display: flex; align-items: center; gap: 6px;
            background: var(--surface2); border: 1px solid var(--border);
            border-radius: 20px; padding: 5px 12px;
            font-size: 0.78rem; font-weight: 500; color: var(--text-muted);
        }
        .badge.ativo { color: var(--green); border-color: var(--green); background: rgba(46,204,113,0.08); }
        .badge.inativo { color: var(--text-muted); }
        .badge .dot { width: 8px; height: 8px; border-radius: 50%; background: currentColor; }
        .badge.ativo .dot { animation: pulse 2s infinite; }
        @keyframes pulse { 0%,100%{opacity:1;}50%{opacity:0.4;} }

        /* LAYOUT PRINCIPAL: Graficos | (Atuadores + Historico) */
        .main-layout {
            display: grid;
            grid-template-columns: 1fr 360px;
            gap: 16px;
            margin-bottom: 20px;
            align-items: start;
        }
        @media (max-width: 900px) {
            .main-layout { grid-template-columns: 1fr; }
        }

        /* Coluna da direita: atuadores em cima, historico embaixo */
        .coluna-direita {
            display: flex;
            flex-direction: column;
            gap: 16px;
            position: sticky;
            top: 20px;
        }

        /* GRID SENSORES */
        .grid-sensores {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
            gap: 16px;
        }
        .sensor-card {
            background: var(--surface);
            border: 1px solid var(--border);
            border-radius: var(--radius);
            padding: 16px;
        }
        .sensor-card.offline { border-color: var(--red); }
        .sensor-header {
            display: flex; justify-content: space-between; align-items: center;
            margin-bottom: 8px;
        }
        .sensor-nome { font-size: 0.8rem; font-weight: 600; color: var(--text-muted); text-transform: uppercase; }
        .sensor-valor { font-size: 1.5rem; font-weight: 700; color: var(--text); }
        .sensor-unidade { font-size: 0.85rem; color: var(--text-muted); }
        .sensor-card canvas { width: 100% !important; height: 80px !important; }

        /* Banner de sensor offline — aparece acima do grafico */
        .sensor-offline-banner {
            display: none;
            background: rgba(231,76,60,0.15);
            border: 1px solid var(--red);
            border-radius: 6px;
            padding: 6px 10px;
            font-size: 0.75rem;
            color: var(--red);
            margin-bottom: 8px;
            text-align: center;
        }
        .sensor-card.offline .sensor-offline-banner { display: block; }

        /* PAINEL GENERICO */
        .painel {
            background: var(--surface);
            border: 1px solid var(--border);
            border-radius: var(--radius);
            padding: 16px;
        }
        .painel-titulo {
            font-size: 0.75rem; font-weight: 700; text-transform: uppercase;
            letter-spacing: 0.08em; color: var(--text-muted);
            margin-bottom: 12px;
        }

        /* ATUADORES */
        .grupo-atuador { margin-bottom: 12px; }
        .grupo-atuador:last-child { margin-bottom: 0; }
        .grupo-label {
            font-size: 0.72rem; font-weight: 600; color: var(--gold);
            margin-bottom: 6px; text-transform: uppercase;
            display: flex; align-items: center; gap: 8px;
        }
        /* Badge de status do atuador (online/offline) ao lado do label */
        .atuador-status {
            font-size: 0.65rem; padding: 1px 6px; border-radius: 4px;
            font-weight: 700;
        }
        .atuador-status.online { background: rgba(46,204,113,0.2); color: var(--green); border: 1px solid var(--green); }
        .atuador-status.offline { background: rgba(231,76,60,0.2); color: var(--red); border: 1px solid var(--red); }

        .grupo-botoes { display: flex; gap: 8px; }
        .btn {
            flex: 1; padding: 8px 10px;
            border: 1px solid var(--border);
            border-radius: 8px;
            font-family: monospace; font-size: 0.78rem; font-weight: 600;
            cursor: pointer; background: var(--surface2); color: var(--text-muted);
        }
        .btn.ligar { border-color: var(--green); color: var(--green); }
        .btn.ligar:hover { background: rgba(46,204,113,0.15); }
        .btn.desligar { border-color: var(--red); color: var(--red); }
        .btn.desligar:hover { background: rgba(231,76,60,0.15); }
        /* Botoes desabilitados quando atuador offline */
        .btn:disabled { opacity: 0.4; cursor: not-allowed; }

        /* HISTORICO DE ACOES */
        .historico-painel {
            background: var(--surface);
            border: 1px solid var(--border);
            border-radius: var(--radius);
            padding: 16px;
            max-height: 340px;
            overflow: hidden;
            display: flex;
            flex-direction: column;
        }
        .acoes-lista {
            flex: 1; overflow-y: auto;
            display: flex; flex-direction: column; gap: 6px;
        }
        .acoes-lista::-webkit-scrollbar { width: 4px; }
        .acoes-lista::-webkit-scrollbar-thumb { background: var(--border); border-radius: 2px; }

        .acao-item {
            padding: 8px 10px;
            background: var(--surface2);
            border: 1px solid var(--border);
            border-radius: 6px;
            font-size: 0.75rem;
        }
        .acao-top { display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px; }
        .acao-hora { color: var(--text-muted); font-size: 0.7rem; }
        .acao-origem {
            font-size: 0.62rem; font-weight: 700; padding: 1px 5px;
            border-radius: 3px;
        }
        .origem-auto { background: rgba(52,152,219,0.2); color: var(--blue); border: 1px solid rgba(52,152,219,0.4); }
        .origem-manual { background: rgba(212,160,83,0.2); color: var(--gold); border: 1px solid rgba(212,160,83,0.4); }
        .acao-desc { color: var(--text); }
        .acao-status-ok { color: var(--green); font-weight: 700; }
        .acao-status-falha { color: var(--red); font-weight: 700; }

        /* ALERTAS */
        .alertas-painel {
            background: var(--surface);
            border: 1px solid var(--border);
            border-radius: var(--radius);
            padding: 16px;
            margin-top: 16px;
        }
        .alerta-item {
            display: flex; align-items: flex-start; gap: 8px;
            padding: 8px 10px; border-radius: 6px; margin-bottom: 6px;
            font-size: 0.78rem;
        }
        .alerta-item:last-child { margin-bottom: 0; }
        .alerta-critico { background: rgba(231,76,60,0.1); border: 1px solid rgba(231,76,60,0.3); }
        .alerta-aviso { background: rgba(243,156,18,0.1); border: 1px solid rgba(243,156,18,0.3); }
        .alerta-hora { color: var(--text-muted); font-size: 0.72rem; min-width: 55px; }
        .alerta-msg { flex: 1; color: var(--text); }

        .sem-dados { color: var(--text-muted); font-size: 0.8rem; padding: 8px 0; }
    </style>
</head>
<body>

<!-- HEADER -->
<header class="header">
    <div class="header-left">
        <div class="logo">🍷</div>
        <div>
            <h1>Vinicola — Painel de Controle</h1>
            <div style="font-size:0.72rem;color:var(--text-muted);margin-top:2px">Monitoramento em Tempo Real</div>
        </div>
    </div>
    <div class="status-badges">
        <div class="badge" id="badge-mosto"><div class="dot"></div> Refrig. Mosto</div>
        <div class="badge" id="badge-adega"><div class="dot"></div> Refrig. Adega</div>
        <div class="badge" id="badge-bomba"><div class="dot"></div> Bomba</div>
    </div>
</header>

<div id="broker-offline">⚠️ CONEXÃO COM O BROKER PERDIDA! Tentando reconectar...</div>

<!-- LAYOUT PRINCIPAL -->
<div class="main-layout">

    <!-- ESQUERDA: Graficos dos sensores -->
    <div class="grid-sensores" id="grid-sensores"></div>

    <!-- DIREITA: Atuadores (em cima) + Historico (embaixo) -->
    <div class="coluna-direita">

        <!-- PAINEL DE ATUADORES -->
        <div class="painel">
            <div class="painel-titulo">Controle de Atuadores</div>

            <div class="grupo-atuador">
                <div class="grupo-label">
                    Resfriamento do Mosto
                    <span class="atuador-status offline" id="status-9090">OFFLINE</span>
                </div>
                <div class="grupo-botoes">
                    <button class="btn ligar" id="btn-ligar-9090" onclick="cmd('9090','LIGAR_REFRIG_MOSTO')" disabled>Ligar</button>
                    <button class="btn desligar" id="btn-desligar-9090" onclick="cmd('9090','DESLIGAR_REFRIG_MOSTO')" disabled>Desligar</button>
                </div>
            </div>

            <div class="grupo-atuador">
                <div class="grupo-label">
                    Resfriamento da Adega
                    <span class="atuador-status offline" id="status-9092">OFFLINE</span>
                </div>
                <div class="grupo-botoes">
                    <button class="btn ligar" id="btn-ligar-9092" onclick="cmd('9092','LIGAR_REFRIG_ADEGA')" disabled>Ligar</button>
                    <button class="btn desligar" id="btn-desligar-9092" onclick="cmd('9092','DESLIGAR_REFRIG_ADEGA')" disabled>Desligar</button>
                </div>
            </div>

            <div class="grupo-atuador">
                <div class="grupo-label">
                    Bomba de Trasfega
                    <span class="atuador-status offline" id="status-9091">OFFLINE</span>
                </div>
                <div class="grupo-botoes">
                    <button class="btn ligar" id="btn-ligar-9091" onclick="cmd('9091','ACIONAR_BOMBA')" disabled>Acionar</button>
                    <button class="btn desligar" id="btn-desligar-9091" onclick="cmd('9091','PARAR_BOMBA')" disabled>Parar</button>
                </div>
            </div>
        </div>

        <!-- HISTORICO DE ACOES -->
        <div class="historico-painel">
            <div class="painel-titulo">Historico de Acoes</div>
            <div class="acoes-lista" id="lista-acoes">
                <div class="sem-dados">Aguardando acoes...</div>
            </div>
        </div>

    </div>
</div>

<!-- ALERTAS -->
<div class="alertas-painel">
    <div class="painel-titulo">Alertas do Sistema</div>
    <div id="lista-alertas"><div class="sem-dados">Nenhum alerta registrado.</div></div>
</div>

<script>
    const graficos = {};
    const unidades = {
        'Temperatura do Mosto': 'C',
        'Densidade do Mosto': '',
        'Umidade da Adega': '%',
        'Temp da Adega': 'C',
        'Nivel da Dorna': '%'
    };
    const coresGraficos = {
        'Temperatura do Mosto': '#e74c3c',
        'Densidade do Mosto': '#8B1A2E',
        'Umidade da Adega': '#3498db',
        'Temp da Adega': '#9b59b6',
        'Nivel da Dorna': '#2ecc71'
    };

    // Atualiza badge do estado do atuador (ligado/desligado)
    function atualizarBadge(id, ligado) {
        const el = document.getElementById(id);
        if (!el) return;
        el.className = 'badge ' + (ligado ? 'ativo' : 'inativo');
    }
    


    // Atualiza o status online/offline dos atuadores e habilita/desabilita botoes
    function atualizarAtuadores(statusAtuadores) {
        if (!statusAtuadores) return;
        const portas = [':9090', ':9091', ':9092'];
        portas.forEach(function(porta) {
            const num = porta.replace(':', '');
            const info = statusAtuadores[porta];
            const online = info && info.online;
            const badge = document.getElementById('status-' + num);
            const btnL = document.getElementById('btn-ligar-' + num);
            const btnD = document.getElementById('btn-desligar-' + num);
            if (badge) {
                badge.textContent = online ? 'ONLINE' : 'OFFLINE';
                badge.className = 'atuador-status ' + (online ? 'online' : 'offline');
            }
            if (btnL) btnL.disabled = !online;
            if (btnD) btnD.disabled = !online;
        });
    }

    // Renderiza os cards dos sensores com banner de offline se necessario
    function renderizarSensores(sensores, statusSensores) {
        const container = document.getElementById('grid-sensores');

        // Conjunto de sensores recebidos agora
        const tiposAtivos = Object.keys(sensores || {});

        // Primeiro, atualiza/cria cards para sensores com dados
        tiposAtivos.forEach(function(sensor) {
            const pontos = sensores[sensor];
            if (!pontos || pontos.length === 0) return;

            const valores = pontos.map(function(p) { return p.v; });
            const timestamps = pontos.map(function(p) {
                const d = new Date(p.t * 1000);
                return d.toTimeString().substring(0, 8);
            });

            const ultimo = valores[valores.length - 1];
            const unidade = unidades[sensor] || '';
            const cor = coresGraficos[sensor] || '#8B1A2E';
            const idSafe = sensor.replace(/\s+/g, '_');

            // Verifica status do sensor
            const info = statusSensores && statusSensores[sensor];
            const online = info ? info.online : true;
            const ultimoSinal = info ? info.ultimo_sinal : '';

            if (!graficos[sensor]) {
                // Cria card novo
                const div = document.createElement('div');
                div.className = 'sensor-card' + (online ? '' : ' offline');
                div.id = 'card-' + idSafe;
                div.innerHTML =
                    '<div class="sensor-offline-banner" id="banner-' + idSafe + '">' +
                        'Sensor Fora de Ar — Ultimo Dado: <span id="ultimosinal-' + idSafe + '">' + ultimoSinal + '</span>' +
                    '</div>' +
                    '<div class="sensor-header">' +
                        '<div class="sensor-nome">' + sensor + '</div>' +
                        '<div>' +
                            '<span class="sensor-valor" id="val-' + idSafe + '">--</span>' +
                            '<span class="sensor-unidade"> ' + unidade + '</span>' +
                        '</div>' +
                    '</div>' +
                    '<canvas id="chart-' + idSafe + '"></canvas>';
                container.appendChild(div);

                const ctx = document.getElementById('chart-' + idSafe).getContext('2d');
                graficos[sensor] = new Chart(ctx, {
                    type: 'line',
                    data: {
                        labels: [],
                        datasets: [{
                            data: [],
                            borderColor: cor,
                            backgroundColor: cor + '22',
                            borderWidth: 2,
                            fill: true,
                            tension: 0.4,
                            pointRadius: 0
                        }]
                    },
                    options: {
                        animation: false,
                        plugins: { legend: { display: false } },
                        scales: {
                            x: { display: false },
                            y: {
                                display: true,
                                grid: { color: '#2d2540' },
                                ticks: { color: '#9b91af', font: { size: 10 }, maxTicksLimit: 4 }
                            }
                        }
                    }
                });
            }

            // Atualiza classe de offline
            const card = document.getElementById('card-' + idSafe);
            if (card) {
                if (online) {
                    card.classList.remove('offline');
                } else {
                    card.classList.add('offline');
                }
            }

            // Atualiza banner de offline
            const banner = document.getElementById('ultimosinal-' + idSafe);
            if (banner) banner.textContent = ultimoSinal;

            // Atualiza valor atual
            const valEl = document.getElementById('val-' + idSafe);
            if (valEl) {
                valEl.textContent = sensor === 'Densidade do Mosto' ? ultimo.toFixed(3) : ultimo.toFixed(1);
            }

            // Atualiza grafico (ultimos 60 pontos)
            graficos[sensor].data.labels = timestamps.slice(-60);
            graficos[sensor].data.datasets[0].data = valores.slice(-60);
            graficos[sensor].update();
        });

        // Atualiza cards de sensores que estao offline mas ja tiveram dados
        if (statusSensores) {
            Object.keys(statusSensores).forEach(function(sensor) {
                const info = statusSensores[sensor];
                if (!info.online) {
                    const idSafe = sensor.replace(/\s+/g, '_');
                    const card = document.getElementById('card-' + idSafe);
                    if (card) {
                        card.classList.add('offline');
                        const banner = document.getElementById('ultimosinal-' + idSafe);
                        if (banner) banner.textContent = info.ultimo_sinal;
                    }
                }
            });
        }
    }

    function renderizarAcoes(acoes) {
        const el = document.getElementById('lista-acoes');
        if (!el) return;
        if (!acoes || acoes.length === 0) {
            el.innerHTML = '<div class="sem-dados">Nenhuma acao registrada.</div>';
            return;
        }
        el.innerHTML = acoes.map(function(a) {
            const origemClass = a.origem === 'AUTO' ? 'origem-auto' : 'origem-manual';
            const statusClass = a.status === 'CONFIRMADO' ? 'acao-status-ok' : 'acao-status-falha';
            const statusIcon = a.status === 'CONFIRMADO' ? '[OK]' : '[FALHA]';
            return '<div class="acao-item">' +
                '<div class="acao-top">' +
                    '<span class="acao-hora">' + a.hora + '</span>' +
                    '<span class="acao-origem ' + origemClass + '">' + a.origem + '</span>' +
                    '<span class="' + statusClass + '">' + statusIcon + '</span>' +
                '</div>' +
                '<div class="acao-desc">' + a.descricao + '</div>' +
            '</div>';
        }).join('');
    }

    function renderizarAlertas(alertas) {
        const el = document.getElementById('lista-alertas');
        if (!alertas || alertas.length === 0) {
            el.innerHTML = '<div class="sem-dados">Nenhum alerta registrado.</div>';
            return;
        }
        el.innerHTML = alertas.map(function(a) {
            const classe = a.nivel === 'CRITICO' ? 'alerta-critico' : 'alerta-aviso';
            const icon = a.nivel === 'CRITICO' ? '[!]' : '[~]';
            return '<div class="alerta-item ' + classe + '">' +
                '<span class="alerta-hora">' + a.hora + '</span>' +
                '<span class="alerta-msg">' + icon + ' ' + a.mensagem + '</span>' +
            '</div>';
        }).join('');
    }
    
    
      
    

    function buscarStatus() {
        fetch('/api/status')
            .then(function(r) { return r.json(); })
            .then(function(data) {
		    const banner = document.getElementById('broker-offline');
		    if (data.broker_online === false) {
		        banner.style.display = 'block';
		        return; // Para a renderização aqui para congelar os ultimos dados na tela
		    } else {
		        banner.style.display = 'none';
		    }
                if (data.sensores) renderizarSensores(data.sensores, data.status_sensores);
                if (data.acoes !== undefined) renderizarAcoes(data.acoes);
                if (data.alertas !== undefined) renderizarAlertas(data.alertas);
                if (data.estado) {
                    atualizarBadge('badge-mosto', data.estado.chiller_mosto);
                    atualizarBadge('badge-adega', data.estado.chiller_adega);
                    atualizarBadge('badge-bomba', data.estado.bomba);
                }
                if (data.status_atuadores) atualizarAtuadores(data.status_atuadores);
            })
            .catch(function() {});
    }

    function cmd(porta, acao) {
        fetch('/api/comando?porta=' + porta + '&acao=' + acao, { method: 'POST' })
            .then(function() { setTimeout(buscarStatus, 500); });
    }

    buscarStatus();
    setInterval(buscarStatus, 1000);
</script>
</body>
</html>`

func main() {
	brokerHost := os.Getenv("BROKER_HOST")
	if brokerHost == "" {
		brokerHost = "172.16.103.1"
	}
	brokerAddr := brokerHost + ":8081"

	ln, err := net.Listen("tcp", ":3000")
	if err != nil {
		fmt.Println("Erro ao iniciar servidor TCP:", err)
		return
	}
	defer ln.Close()

	fmt.Println("Painel disponivel em http://localhost:3000")

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go tratarConexaoHTTP(conn, brokerAddr)
	}
}

func tratarConexaoHTTP(conn net.Conn, brokerAddr string) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	linha, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	// Consome os headers HTTP ate a linha vazia
	for {
		header, err := reader.ReadString('\n')
		if err != nil || header == "\r\n" {
			break
		}
	}

	partes := strings.Fields(strings.TrimSpace(linha))
	if len(partes) < 2 {
		return
	}

	metodo := partes[0]
	caminhoCompleto := partes[1]

	caminho := caminhoCompleto
	queryString := ""
	if idx := strings.Index(caminhoCompleto, "?"); idx != -1 {
		caminho = caminhoCompleto[:idx]
		queryString = caminhoCompleto[idx+1:]
	}

	switch {
	case caminho == "/" || caminho == "":
		enviarResposta(conn, "200 OK", "text/html; charset=utf-8", htmlPagina)

	case caminho == "/api/status":
		resposta := comunicarComBroker("GET_STATUS", brokerAddr)
		enviarResposta(conn, "200 OK", "application/json", resposta)

	case caminho == "/api/comando" && metodo == "POST":
		params := parseQuery(queryString)
		porta := params["porta"]
		acao := params["acao"]
		comunicarComBroker(fmt.Sprintf("CMD:%s:%s", porta, acao), brokerAddr)
		enviarResposta(conn, "200 OK", "application/json", "{\"ok\":true}")

	default:
		enviarResposta(conn, "404 Not Found", "text/plain", "404 - Nao encontrado")
	}
}

func enviarResposta(conn net.Conn, status string, contentType string, corpo string) {
	resposta := fmt.Sprintf("HTTP/1.1 %s\r\nContent-Type: %s\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s",
		status, contentType, len(corpo), corpo)
	conn.Write([]byte(resposta))
}

func parseQuery(query string) map[string]string {
	resultado := make(map[string]string)
	if query == "" {
		return resultado
	}
	pares := strings.Split(query, "&")
	for _, par := range pares {
		kv := strings.SplitN(par, "=", 2)
		if len(kv) == 2 {
			resultado[kv[0]] = kv[1]
		}
	}
	return resultado
}


func comunicarComBroker(mensagem, addr string) string {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return "{\"sensores\":{},\"acoes\":[],\"alertas\":[],\"estado\":{},\"status_sensores\":{},\"status_atuadores\":{}}"
	}
	defer conn.Close()
	fmt.Fprintln(conn, mensagem)
	resposta, _ := bufio.NewReader(conn).ReadString('\n')
	return strings.TrimSpace(resposta)
}
