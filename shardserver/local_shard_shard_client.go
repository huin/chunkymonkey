package shardserver

import (
	"github.com/huin/chunkymonkey/gamerules"
	. "github.com/huin/chunkymonkey/types"
)

// localShardShardClient implements IShardShardClient for LocalShardManager.
type localShardShardClient struct {
	serverShard *ChunkShard
}

func newLocalShardShardClient(serverShard *ChunkShard) *localShardShardClient {
	return &localShardShardClient{
		serverShard: serverShard,
	}
}

func (client *localShardShardClient) Disconnect() {
}

func (client *localShardShardClient) ReqSetActiveBlocks(blocks []BlockXyz) {
	client.serverShard.enqueue(func() {
		client.serverShard.reqSetBlocksActive(blocks)
	})
}

func (client *localShardShardClient) ReqTransferEntity(loc ChunkXz, entity gamerules.INonPlayerEntity) {
	client.serverShard.enqueue(func() {
		chunk := client.serverShard.chunkAt(loc)
		if chunk != nil {
			chunk.transferEntity(entity)
		}
	})
}
