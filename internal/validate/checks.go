package validate

import (
	"github.com/afelin/curbpack/internal/ir"
	"github.com/afelin/curbpack/internal/packs"
)

type CheckKind string

const (
	CheckAnnexFile       CheckKind = "annex_file"
	CheckFilePresent     CheckKind = "file_present"
	CheckAntiPlaceholder CheckKind = "anti_placeholder"
	CheckNPMDepBan       CheckKind = "npm_dep_ban"
	CheckManifestDepBan  CheckKind = "manifest_dep_ban"
	CheckTextForbid      CheckKind = "text_forbid"
	CheckImportReach     CheckKind = "import_reach"
	CheckFresh           CheckKind = "fresh"
	CheckOwned           CheckKind = "owned"
)

type checkFn func(root string, rule packs.Rule) []ir.Failure

var checkRegistry = map[CheckKind]checkFn{
	CheckAnnexFile:       checkFilePresent,
	CheckFilePresent:     checkFilePresent,
	CheckAntiPlaceholder: checkAntiPlaceholder,
	CheckNPMDepBan:       checkNPMDepBan,
	CheckManifestDepBan:  checkNPMDepBan,
	CheckTextForbid:      checkTextForbid,
	CheckImportReach:     func(root string, rule packs.Rule) []ir.Failure { return auditASTReachability(root) },
	CheckFresh:           checkFresh,
	CheckOwned:           checkOwned,
}

func KnownCheckKinds() []CheckKind {
	out := make([]CheckKind, 0, len(checkRegistry))
	for k := range checkRegistry {
		out = append(out, k)
	}
	return out
}
