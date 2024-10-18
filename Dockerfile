FROM ubuntu:latest

RUN mkdir /app
COPY ./stash /app
WORKDIR /app
EXPOSE 5555

ENTRYPOINT ["./stash"]
