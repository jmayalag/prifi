package prifi

import (
	"github.com/dedis/cothority/sda"
	prifi_lib "github.com/lbarman/prifi_dev/prifi-lib"
)

/*
This file defines messages that can be handled by the SDA
and containing the various messages used by the PriFi library.
 */

// Wrapper to be able to send message through SDA.
type Struct_ALL_ALL_SHUTDOWN struct {
	*sda.TreeNode
	prifi_lib.ALL_ALL_SHUTDOWN
}
// Wrapper to be able to send message through SDA.
type Struct_ALL_ALL_PARAMETERS struct {
	*sda.TreeNode
	prifi_lib.ALL_ALL_PARAMETERS
}
// Wrapper to be able to send message through SDA.
type Struct_CLI_REL_TELL_PK_AND_EPH_PK struct {
	*sda.TreeNode
	prifi_lib.CLI_REL_TELL_PK_AND_EPH_PK
}
// Wrapper to be able to send message through SDA.
type Struct_CLI_REL_UPSTREAM_DATA struct {
	*sda.TreeNode
	prifi_lib.CLI_REL_UPSTREAM_DATA
}
// Wrapper to be able to send message through SDA.
type Struct_REL_CLI_DOWNSTREAM_DATA struct {
	*sda.TreeNode
	prifi_lib.REL_CLI_DOWNSTREAM_DATA
}
// Wrapper to be able to send message through SDA.
type Struct_REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG struct {
	*sda.TreeNode
	prifi_lib.REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG
}
// Wrapper to be able to send message through SDA.
type Struct_REL_CLI_TELL_TRUSTEES_PK struct {
	*sda.TreeNode
	prifi_lib.REL_CLI_TELL_TRUSTEES_PK
}
// Wrapper to be able to send message through SDA.
type Struct_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE struct {
	*sda.TreeNode
	prifi_lib.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE
}
// Wrapper to be able to send message through SDA.
type Struct_REL_TRU_TELL_TRANSCRIPT struct {
	*sda.TreeNode
	prifi_lib.REL_TRU_TELL_TRANSCRIPT
}
// Wrapper to be able to send message through SDA.
type Struct_TRU_REL_DC_CIPHER struct {
	*sda.TreeNode
	prifi_lib.TRU_REL_DC_CIPHER
}
// Wrapper to be able to send message through SDA.
type Struct_TRU_REL_SHUFFLE_SIG struct {
	*sda.TreeNode
	prifi_lib.TRU_REL_SHUFFLE_SIG
}
// Wrapper to be able to send message through SDA.
type Struct_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS struct {
	*sda.TreeNode
	prifi_lib.TRU_REL_TELL_NEW_BASE_AND_EPH_PKS
}
// Wrapper to be able to send message through SDA.
type Struct_TRU_REL_TELL_PK struct {
	*sda.TreeNode
	prifi_lib.TRU_REL_TELL_PK
}
// Wrapper to be able to send message through SDA.
type Struct_REL_TRU_TELL_RATE_CHANGE struct {
	*sda.TreeNode
	prifi_lib.REL_TRU_TELL_RATE_CHANGE
}
