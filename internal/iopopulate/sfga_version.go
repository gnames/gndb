package iopopulate

import (
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gnlib"
)

func (p *populator) checkSfgaVersion(sourceID int) error {
	var version string
	row := p.sfgaDB.QueryRow("SELECT ID FROM VERSION LIMIT 1")
	err := row.Scan(&version)
	if err != nil {
		return SfgaGetVersionError(sourceID, err)
	}
	if !gnlib.IsVersion(version) {
		return NotSfgaVersion(sourceID, version)
	}

	if gnlib.CmpVersion(version, config.MinVersionSFGA) < 0 {
		return SFGAVersionTooOld(sourceID, version)
	}
	return nil
}
