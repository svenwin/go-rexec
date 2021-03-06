package main

import (
  "io"
  "log"
  "net"
  "os"

  "github.com/docker/libchan"
  "github.com/docker/libchan/spdy"
)

type RemoteCommand struct {
  Cmd        string
  Args       []string
  Stdin      io.Writer
  Stdout     io.Reader
  Stderr     io.Reader
  StatusChan libchan.Sender
}

type CommandResponse struct {
  Status int
}

func main() {
  if len(os.Args) < 2 {
    log.Fatal("usage: <command> [<arg> ]")
  }

  var client net.Conn
  var err error

  port := os.Getenv("SBP")

  client, err = net.Dial("tcp", "127.0.0.1:" + port)

  if err != nil {
    log.Fatal(err)
  }

  transport, err := spdy.NewClientTransport(client)
  if err != nil {
    log.Fatal(err)
  }
  sender, err := transport.NewSendChannel()
  if err != nil {
    log.Fatal(err)
  }

  receiver, remoteSender := libchan.Pipe()

  command := &RemoteCommand{
    Cmd:        os.Args[1],
    Args:       os.Args[2:],
    Stdin:      os.Stdin,
    Stdout:     os.Stdout,
    Stderr:     os.Stderr,
    StatusChan: remoteSender,
  }

  err = sender.Send(command)
  if err != nil {
    log.Fatal(err)
  }

  response := &CommandResponse{}
  err = receiver.Receive(response)
  if err != nil {
    log.Fatal(err)
  }

  os.Exit(response.Status)
}
