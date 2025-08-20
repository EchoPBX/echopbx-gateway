package sdk

// Plugin es la interfaz que todos los plugins deben implementar.
// El gateway la usará para cargar, iniciar y detener cada plugin.
type Plugin interface {
	// Init se llama cuando el plugin se carga.
	// Recibe un Context para loggear, publicar eventos, etc.
	Init(ctx *Context) error

	// Stop se llama cuando el gateway se apaga o recarga los plugins.
	Stop() error
}
