package config

import conf "config/context"

type skipSectionMode int8

const (
	startSkipping skipSectionMode = iota
	stopSkipping
)

type unMarshaller interface {
	scan() bool
	prepareData(*conf.Context, *Config) error
	getDataStrategy(*conf.Context) dataStrategy
	setSkipSectionMode(skipSectionMode)
	getFileRowData() *fileRowData
}

func unMarshall(ctx *conf.Context, um unMarshaller, conf *Config, logger *Logger) error {

	// Get next data
	for um.scan() {

		// Do some preprocessing if needed
		if err := um.prepareData(ctx, conf); err != nil {
			return err
		}

		// get appropriate dataStrategy
		ds := um.getDataStrategy(ctx)

		// execute strategy
		if err := ds.execute(ctx, conf, um, logger); err != nil {
			return err
		}
	}
	return nil
}
