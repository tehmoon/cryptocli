package flags

import (
	"../codec"
	"fmt"
	"flag"
	"os"
	"github.com/tehmoon/errors"
	"strings"
	"../inout"
	"../filter"
)

type GlobalFlags struct {
	Chomp bool
	Encoders string
	Decoders string
	In string
	Out string
	FiltersIn string
	FiltersCmdIn string
	FiltersCmdOut string
	FiltersOut string
	TeeIn string
	TeeCmdIn string
	TeeCmdOut string
	TeeOut string
}

var ErrBadFlag = errors.New("Bad flags\n")

func ParseFlags(set *flag.FlagSet, globalFlags *GlobalFlags) (*GlobalOptions) {
	globalOptions := newGlobalOptions()

	set.Parse(os.Args[2:])

	var err error

	if globalFlags.FiltersIn != "" {
		globalOptions.FiltersIn, err = filter.ParseAll(globalFlags.FiltersIn)
		if err != nil {
			fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error parsing -filters-in").Error())
			os.Exit(2)
		}
	}

	if globalFlags.FiltersCmdIn != "" {
		globalOptions.FiltersCmdIn, err = filter.ParseAll(globalFlags.FiltersCmdIn)
		if err != nil {
			fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error parsing -filters-cmd-in").Error())
			os.Exit(2)
		}
	}

	if globalFlags.FiltersCmdOut != "" {
		globalOptions.FiltersCmdOut, err = filter.ParseAll(globalFlags.FiltersCmdOut)
		if err != nil {
			fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error parsing -filters-cmd-out").Error())
			os.Exit(2)
		}
	}

	if globalFlags.FiltersOut != "" {
		globalOptions.FiltersOut, err = filter.ParseAll(globalFlags.FiltersOut)
		if err != nil {
			fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error parsing -filters-out").Error())
			os.Exit(2)
		}
	}

	if globalFlags.TeeIn != "" {
		globalOptions.TeeIn, err = inout.ParseOutput(globalFlags.TeeIn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			os.Exit(2)
		}
	}
	if globalFlags.TeeCmdIn != "" {
		globalOptions.TeeCmdIn, err = inout.ParseOutput(globalFlags.TeeCmdIn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			os.Exit(2)
		}
	}
	if globalFlags.TeeCmdOut != "" {
		globalOptions.TeeCmdOut, err = inout.ParseOutput(globalFlags.TeeCmdOut)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			os.Exit(2)
		}
	}
	if globalFlags.TeeOut != "" {
		globalOptions.TeeOut, err = inout.ParseOutput(globalFlags.TeeOut)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			os.Exit(2)
		}
	}

	globalOptions.Input, err = inout.ParseInput(globalFlags.In)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(2)
	}

	globalOptions.Output, err = inout.ParseOutput(globalFlags.Out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(2)
	}

	if globalFlags.Decoders != "" {
		cvs, err := codec.ParseAll(strings.Split(globalFlags.Decoders, ","))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing decoders. Err: %v", err)
			os.Exit(2)
		}

		decoders := make([]codec.CodecDecoder, len(cvs))
		for i, cv := range cvs {
			dec := cv.Codec.Decoder(cv.Values)
			if dec == nil {
				fmt.Fprintf(os.Stderr, "Codec %s doesn't support decoding\n", cv.Codec.Name())
				os.Exit(2)
			}

			decoders[i] = dec
		}

		globalOptions.Decoders = decoders
	}

	if globalFlags.Encoders != "" {
		cvs, err := codec.ParseAll(strings.Split(globalFlags.Encoders, ","))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing encoders. Err: %v", err)
			os.Exit(2)
		}

		encoders := make([]codec.CodecEncoder, len(cvs))
		for i, cv := range cvs {
			enc := cv.Codec.Encoder(cv.Values)
			if enc == nil {
				fmt.Fprintf(os.Stderr, "Codec %s doesn't support encoding\n", cv.Codec.Name())
				os.Exit(2)
			}

			encoders[i] = enc
		}

		globalOptions.Encoders = encoders
	}

	globalOptions.Chomp = globalFlags.Chomp

	return globalOptions
}

func SetupFlags(set *flag.FlagSet) (*GlobalFlags) {
	globalFlags := &GlobalFlags{}

	set.BoolVar(&globalFlags.Chomp, "chomp", false, "Get rid of the last \\n when not in pipe")
	set.StringVar(&globalFlags.Decoders, "decoders", "binary", "Set a list of codecs separated by ',' to decode input that will be process in the order given")
	set.StringVar(&globalFlags.Encoders, "encoders", "binary", "Set a list of codecs separated by ',' to encode output that will be process in the order given")
	set.StringVar(&globalFlags.In, "in", "", "Input <fileType> method")
	set.StringVar(&globalFlags.Out, "out", "", "Output <fileType> method")
	set.StringVar(&globalFlags.TeeIn, "tee-in", "", "Copy output before -encoders to <fileType>")
	set.StringVar(&globalFlags.TeeCmdIn, "tee-cmd-in", "", "Copy output after -decoders and before <command> to <fileType>")
	set.StringVar(&globalFlags.TeeCmdOut, "tee-cmd-out", "", "Copy output after <command> and before -encoders to <fileType>")
	set.StringVar(&globalFlags.TeeOut, "tee-out", "", "Copy output after -encoders to <fileType>")
	set.StringVar(&globalFlags.FiltersIn, "filters-in", "", "List of <filter> in URL format that filters data right before -decoders")
	set.StringVar(&globalFlags.FiltersCmdIn, "filters-cmd-in", "", "List of <filter> in URL format that filters data right after -decoders")
	set.StringVar(&globalFlags.FiltersCmdOut, "filters-cmd-out", "", "List of <filter> in URL format that filters data right before -encoders")
	set.StringVar(&globalFlags.FiltersOut, "filters-out", "", "List of <filter> in URL format that filters data right after -encoders")

	return globalFlags
}
