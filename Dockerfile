FROM ubuntu:latest
RUN apt update
RUN apt install -y wget
RUN wget https://go.dev/dl/go1.20.4.linux-amd64.tar.gz
RUN tar -xvzf ./go1.20.4.linux-amd64.tar.gz -C /usr/local/
RUN export PATH=$PATH:/usr/local/go/bin
COPY . sointu
WORKDIR /sointu
ENV PATH="$PATH:/usr/local/go/bin"
RUN go build -o sointu-server cmd/sointu-compile/server.go
CMD ./sointu-server