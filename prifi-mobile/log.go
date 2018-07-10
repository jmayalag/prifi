package prifiMobile

import "gopkg.in/dedis/onet.v2/log"

// Functions to setup logging

type PrifiLogger interface {
	Log(level int, msg string)
	Close()
}

type PrifiLogging struct {
	logger PrifiLogger
	lInfo  *log.LoggerInfo
}

func (pl *PrifiLogging) Log(level int, msg string) {
	pl.logger.Log(level, msg)
}

func (pl *PrifiLogging) Close() {
	pl.logger.Close()
}

func (pl *PrifiLogging) GetLoggerInfo() *log.LoggerInfo {
	return pl.lInfo
}

func NewPrifiLogging(debugLvl int, showTime bool, useColors bool, padding bool, l PrifiLogger) *PrifiLogging {
	lInfo := &log.LoggerInfo{
		DebugLvl:  debugLvl,
		ShowTime:  showTime,
		UseColors: useColors,
		Padding:   padding,
	}

	return &PrifiLogging{
		logger: l,
		lInfo:  lInfo,
	}
}

func RegisterPrifiLogging(pl *PrifiLogging) int {
	return log.RegisterLogger(pl)
}

func LogInfo(msg string) {
	log.Info(msg)
}
