package stencil

import "io"

func WithAppName(name string) Option {
	return func(s *Stencil) {
		s.AppName = name
	}
}

func WithAppDescription(desc string) Option {
	return func(s *Stencil) {
		s.AppDesc = desc
	}
}

func WithVersion(show bool) Option {
	return func(s *Stencil) {
		s.ShowVersion = show
	}
}

func WithBuildDate(show bool) Option {
	return func(s *Stencil) {
		s.ShowBuildDate = show
	}
}

func WithCommitHash(show bool) Option {
	return func(s *Stencil) {
		s.ShowCommitHash = show
	}
}

func WithBranch(show bool) Option {
	return func(s *Stencil) {
		s.ShowBranch = show
	}
}

func WithColor(colored bool) Option {
	return func(s *Stencil) {
		s.ColoredOutput = colored
	}
}

func WithOutput(output io.Writer) Option {
	return func(s *Stencil) {
		s.Output = output
	}
}
