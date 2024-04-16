import os
import time
import subprocess
import psutil

def execute_test(rule_name, command, repetitions=100000, sampling_interval=0.1):
    print(f"Testando a regra '{rule_name}'...")
    
    # Captura as métricas de sistema antes de iniciar o teste
    cpu_percent_before = psutil.cpu_percent(interval=sampling_interval)
    memory_info_before = psutil.virtual_memory()
    
    start_time = time.time()
    
    # Executa o comando repetidamente para gerar eventos
    for _ in range(repetitions):
        subprocess.run(command, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    
    end_time = time.time()
    
    # Captura as métricas de sistema após o teste
    cpu_percent_after = psutil.cpu_percent(interval=sampling_interval)
    memory_info_after = psutil.virtual_memory()
    
    # Calcula o tempo de execução e a diferença no uso de CPU e memória
    execution_time = end_time - start_time
    cpu_usage = cpu_percent_after - cpu_percent_before
    memory_usage = (memory_info_after.used - memory_info_before.used) / (1024 * 1024)
    
    print(f"Teste '{rule_name}' concluído.")
    print(f"Tempo de execução: {execution_time:.8f} segundos")
    print(f"Uso de CPU: {cpu_usage:.8f}%")
    print(f"Uso de Memória: {memory_usage:.8f} MB\n")

if __name__ == "__main__":
    execute_test("Terminal in Container", "echo 'Teste Terminal in Container'")
    execute_test("Write below etc", "echo 'Teste Write below etc'")
