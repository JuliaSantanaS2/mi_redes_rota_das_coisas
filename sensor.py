import socket
import time
import random
import os

def iniciar():
    # Puxa o nome correto do servidor do docker-compose
    host = os.getenv('HOST_SERVIDOR', 'servidor_app')
    port = 5000
    
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    id_sensor = ["SENS_01","SENS_02","SENS_03"]
    
    print(f"[SENSOR {id_sensor}] Iniciado. Enviando para {host}:{port}")
    
    while True:
        try:
            valor_01 = random.randint(1, 100)
            valor_02 = random.randint(4, 400) / 4.0

            msg_01 = f"SENSOR|{id_sensor[0]}|{valor_01}"
            msg_02 = f"SENSOR|{id_sensor[1]}|{valor_02}"
            
            sock.sendto(msg_01.encode('utf-8'), (host, port))
            print(f"Enviado: {msg_01}")

            sock.sendto(msg_02.encode('utf-8'), (host, port))
            print(f"Enviado: {msg_02}")
            
            time.sleep(1) 
        except Exception as e:
            print(f"[ERRO SENSOR] Falha ao enviar dado: {e}")
            time.sleep(2)

if __name__ == "__main__":
    iniciar()