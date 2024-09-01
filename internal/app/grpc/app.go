package grpcapp

type App struct {
	port int // change port to GRPCServerOpts ?
}

func New() *App {
	return &App{}
}

func (a *App) Run() {
	//server := grpc.NewServer()

}
