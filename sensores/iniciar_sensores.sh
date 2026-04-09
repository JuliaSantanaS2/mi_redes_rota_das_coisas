#!/bin/bash

# -------------------------------------------------------
# CONFIGURACAO — edite o IP do servidor Broker aqui
# -------------------------------------------------------
SERVER_IP="172.16.103.1:8080"

SENSORES=(
    "sensor-temp-mosto"
    "sensor-densidade"
    "sensor-umidade-adega"
    "sensor-temp-adega"
    "sensor-nivel-dorna"
)

NOMES=(
    "Temperatura do Mosto"
    "Densidade do Mosto"
    "Umidade da Adega"
    "Temperatura da Adega"
    "Nivel da Dorna"
)

LOG_DIR="./logs"
mkdir -p "$LOG_DIR"

echo ""
echo " ================================================"
echo "  Vinicola - Sensores Remotos"
echo " ================================================"
echo ""

if ! docker info > /dev/null 2>&1; then
    echo " [ERRO] Docker nao encontrado ou nao esta rodando."
    exit 1
fi

echo " Servidor Broker: $SERVER_IP"
echo ""
echo " Parando containers anteriores..."
SERVER_IP="$SERVER_IP" docker-compose down > /dev/null 2>&1
sleep 1

# -------------------------------------------------------
# Tenta tmux (melhor experiencia — janelas no mesmo terminal)
# -------------------------------------------------------
if command -v tmux &> /dev/null; then
    echo " Usando tmux — abrindo painel com todos os sensores..."
    sleep 1

    SESSION="vinicola"
    tmux kill-session -t "$SESSION" 2>/dev/null

    tmux new-session -d -s "$SESSION" -x 220 -y 50 \
        -n "${NOMES[0]}" \
        "SERVER_IP=$SERVER_IP docker-compose up --build ${SENSORES[0]}; read"

    for i in 1 2 3 4; do
        tmux new-window -t "$SESSION" -n "${NOMES[$i]}" \
            "SERVER_IP=$SERVER_IP docker-compose up --build ${SENSORES[$i]}; read"
        sleep 1
    done

    tmux attach-session -t "$SESSION"

# -------------------------------------------------------
# Tenta xterm
# -------------------------------------------------------
elif command -v xterm &> /dev/null; then
    echo " Usando xterm — abrindo terminal por sensor..."
    for i in 0 1 2 3 4; do
        xterm -title "${NOMES[$i]}" -fa 'Monospace' -fs 10 \
            -e bash -c "SERVER_IP=$SERVER_IP docker-compose up --build ${SENSORES[$i]}; echo ''; echo 'Pressione ENTER para fechar'; read" &
        sleep 2
    done
    echo ""
    echo " 5 terminais xterm abertos, um por sensor."
    echo " Para parar tudo: docker-compose down"

# -------------------------------------------------------
# Fallback: background com logs em arquivo
# -------------------------------------------------------
else
    echo " Nenhum terminal grafico disponivel."
    echo " Rodando sensores em background — logs em ./logs/"
    echo ""

    for i in 0 1 2 3 4; do
        LOG="$LOG_DIR/${SENSORES[$i]}.log"
        SERVER_IP="$SERVER_IP" docker-compose up --build "${SENSORES[$i]}" > "$LOG" 2>&1 &
        echo " [OK] ${NOMES[$i]} — PID $! — log: $LOG"
        sleep 2
    done

    echo ""
    echo " Acompanhe os logs com:"
    echo "   tail -f logs/sensor-temp-mosto.log"
    echo "   tail -f logs/sensor-densidade.log"
    echo "   tail -f logs/sensor-umidade-adega.log"
    echo "   tail -f logs/sensor-temp-adega.log"
    echo "   tail -f logs/sensor-nivel-dorna.log"
    echo ""
    echo " Para parar tudo: docker-compose down"
fi
