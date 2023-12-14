FROM actlab.azurecr.io/repro_base:latest

WORKDIR /app

ADD entrypoint.sh ./

RUN chmod +x ./entrypoint.sh

ADD ./one-click-aks-server ./
ADD /tf ./tf
ADD /scripts ./scripts
ADD /caddy/Caddyfile /etc/caddy/Caddyfile

EXPOSE 8080/tcp
EXPOSE 443/tcp

ENTRYPOINT [ "/bin/bash", "/app/entrypoint.sh" ]