package options

type Options struct {
	FilePrefix string
}

func ReadOptions() *Options {
	//TODO Read options from protoc command line

	return &Options{
		FilePrefix: "svc-",
	}
}
