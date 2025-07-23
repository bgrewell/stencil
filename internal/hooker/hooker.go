package hooker

type Hooker interface {
	Install(rootPath string) error
	Uninstall(rootPath string) error
}
