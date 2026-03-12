import socket
import threading
import json
import time

buffer_memoria = []
lock = threading.Lock()

def salvar_historico():
    while True:
        time.sleep(5)
        with lock:
            if len(buffer_memoria) > 0:
                dados_salvar = buffer_memoria.copy()
                buffer_memoria.clear()
                try:
                    with open("historico.txt", "a") as f:
                        for d in dados_salvar:
                            f.write(json.dumps(d) + "\n")
                    print(f"[SISTEMA] {len(dados_salvar)} registros salvos no historico.")
                except Exception as e:
                    print(f"[ERRO] Falha ao salvar arquivo: {e}")

def udp_sensores(host, port):
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.bind((host, port))
        print(f"[UDP] Escutando sensores na porta {port}...")
        
        while True:
            data, addr = sock.recvfrom(1024)
            msg = data.decode('utf-8')
            partes = msg.split('|')
            if len(partes) == 3 and partes[0] == "SENSOR":
                registro = {"id": partes[1], "valor": partes[2], "timestamp": time.time()}
                with lock:
                    buffer_memoria.append(registro)
    except Exception as e:
        print(f"[ERRO UDP] {e}")

def tcp_clientes(host, port):
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.bind((host, port))
        sock.listen(5)
        print(f"[TCP] Escutando clientes na porta {port}...")
        
        while True:
            conn, addr = sock.accept()
            threading.Thread(target=tratar_cliente, args=(conn, addr)).start()
    except Exception as e:
        print(f"[ERRO TCP] {e}")

def tratar_cliente(conn, addr):
    print(f"[TCP] Cliente conectado: {addr}")
    try:
        while True:
            data = conn.recv(1024)
            if not data:
                break
            comando = data.decode('utf-8')
            
            if comando == "RELATORIO":
                try:
                    with open("historico.txt", "r") as f:
                        linhas = f.readlines()
                    resposta = f"Total de leituras salvas: {len(linhas)}"
                except FileNotFoundError:
                    resposta = "Nenhum dado salvo ainda."
                conn.send(resposta.encode('utf-8'))
            elif comando.startswith("COMANDO|"):
                _, acao = comando.split('|')
                print(f"[ATUADOR] Executando acao: {acao}")
                conn.send(f"Acao '{acao}' executada.".encode('utf-8'))
            else:
                conn.send("Comando invalido.".encode('utf-8'))
    except Exception as e:
        pass
    finally:
        conn.close()

if __name__ == "__main__":
    threading.Thread(target=udp_sensores, args=('0.0.0.0', 5000), daemon=True).start()
    threading.Thread(target=tcp_clientes, args=('0.0.0.0', 8080), daemon=True).start()
    threading.Thread(target=salvar_historico, daemon=True).start()
    while True:
        time.sleep(1)