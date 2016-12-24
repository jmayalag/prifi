package protocols

import (
	"github.com/dedis/cothority/sda"
	prifi_lib "github.com/lbarman/prifi/prifi-lib"
)

//Struct_ALL_ALL_SHUTDOWN is a wrapper for ALL_ALL_SHUTDOWN (but also contains a *sda.TreeNode)
type Struct_ALL_ALL_SHUTDOWN struct {
	*sda.TreeNode
	prifi_lib.ALL_ALL_SHUTDOWN
}

//Struct_ALL_ALL_PARAMETERS is a wrapper for ALL_ALL_PARAMETERS (but also contains a *sda.TreeNode)
type Struct_ALL_ALL_PARAMETERS struct {
	*sda.TreeNode
	prifi_lib.ALL_ALL_PARAMETERS
}

//Struct_CLI_REL_TELL_PK_AND_EPH_PK is a wrapper for CLI_REL_TELL_PK_AND_EPH_PK (but also contains a *sda.TreeNode)
type Struct_CLI_REL_TELL_PK_AND_EPH_PK struct {
	*sda.TreeNode
	prifi_lib.CLI_REL_TELL_PK_AND_EPH_PK
}

//Struct_CLI_REL_UPSTREAM_DATA is a wrapper for CLI_REL_UPSTREAM_DATA (but also contains a *sda.TreeNode)
type Struct_CLI_REL_UPSTREAM_DATA struct {
	*sda.TreeNode
	prifi_lib.CLI_REL_UPSTREAM_DATA
}

//Struct_REL_CLI_DOWNSTREAM_DATA is a wrapper for REL_CLI_DOWNSTREAM_DATA (but also contains a *sda.TreeNode)
type Struct_REL_CLI_DOWNSTREAM_DATA struct {
	*sda.TreeNode
	prifi_lib.REL_CLI_DOWNSTREAM_DATA
}

//Struct_REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG is a wrapper for REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG (but also contains a *sda.TreeNode)
type Struct_REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG struct {
	*sda.TreeNode
	prifi_lib.REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG
}

//Struct_REL_CLI_TELL_TRUSTEES_PK is a wrapper for REL_CLI_TELL_TRUSTEES_PK (but also contains a *sda.TreeNode)
type Struct_REL_CLI_TELL_TRUSTEES_PK struct {
	*sda.TreeNode
	prifi_lib.REL_CLI_TELL_TRUSTEES_PK
}

//Struct_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE is a wrapper for REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE (but also contains a *sda.TreeNode)
type Struct_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE struct {
	*sda.TreeNode
	prifi_lib.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE
}

//Struct_REL_TRU_TELL_TRANSCRIPT is a wrapper for REL_TRU_TELL_TRANSCRIPT (but also contains a *sda.TreeNode)
type Struct_REL_TRU_TELL_TRANSCRIPT struct {
	*sda.TreeNode
	prifi_lib.REL_TRU_TELL_TRANSCRIPT
}

//Struct_TRU_REL_DC_CIPHER is a wrapper for TRU_REL_DC_CIPHER (but also contains a *sda.TreeNode)
type Struct_TRU_REL_DC_CIPHER struct {
	*sda.TreeNode
	prifi_lib.TRU_REL_DC_CIPHER
}

//Struct_TRU_REL_SHUFFLE_SIG is a wrapper for TRU_REL_SHUFFLE_SIG (but also contains a *sda.TreeNode)
type Struct_TRU_REL_SHUFFLE_SIG struct {
	*sda.TreeNode
	prifi_lib.TRU_REL_SHUFFLE_SIG
}

//Struct_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS is a wrapper for TRU_REL_TELL_NEW_BASE_AND_EPH_PKS (but also contains a *sda.TreeNode)
type Struct_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS struct {
	*sda.TreeNode
	prifi_lib.TRU_REL_TELL_NEW_BASE_AND_EPH_PKS
}

//Struct_TRU_REL_TELL_PK is a wrapper for TRU_REL_TELL_PK (but also contains a *sda.TreeNode)
type Struct_TRU_REL_TELL_PK struct {
	*sda.TreeNode
	prifi_lib.TRU_REL_TELL_PK
}

//Struct_REL_TRU_TELL_RATE_CHANGE is a wrapper for REL_TRU_TELL_RATE_CHANGE (but also contains a *sda.TreeNode)
type Struct_REL_TRU_TELL_RATE_CHANGE struct {
	*sda.TreeNode
	prifi_lib.REL_TRU_TELL_RATE_CHANGE
}
