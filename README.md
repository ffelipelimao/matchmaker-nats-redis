# Primeira vez ou após mudanças
make up-build

# Durante desenvolvimento
make logs-api       # Ver logs da API
make logs-worker    # Ver logs do Worker

# Após mudanças no código
make down-clean
make build && make restart

# Limpeza completa
make clean