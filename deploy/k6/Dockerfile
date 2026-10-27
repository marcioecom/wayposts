# Imagem do gerador de carga. Os cenários vão embutidos para que o Job rode
# dentro do cluster, sem depender de volume ou de kubectl copy.
FROM grafana/k6:latest

COPY load /load
WORKDIR /load

ENTRYPOINT ["k6"]
