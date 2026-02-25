FROM actlabs.azurecr.io/actlabs-base:20260224-02

WORKDIR /app

ADD entrypoint.sh ./

RUN chmod +x ./entrypoint.sh

ADD ./azurerm-msi-auth-proxy ./ 
ADD ./one-click-aks-server ./
ADD /tf ./tf
ADD /scripts ./scripts

EXPOSE 8080/tcp
EXPOSE 443/tcp

ENTRYPOINT [ "/bin/bash", "/app/entrypoint.sh" ]