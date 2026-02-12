import request from '@/utils/request'

// ===== 中间件管理 =====

export const getMiddlewareList = (params: any) => {
  return request.get('/api/v1/middlewares', { params })
}

export const getMiddleware = (id: number) => {
  return request.get(`/api/v1/middlewares/${id}`)
}

export const createMiddleware = (data: any) => {
  return request.post('/api/v1/middlewares', data)
}

export const updateMiddleware = (id: number, data: any) => {
  return request.put(`/api/v1/middlewares/${id}`, data)
}

export const deleteMiddleware = (id: number) => {
  return request.delete(`/api/v1/middlewares/${id}`)
}

export const batchDeleteMiddlewares = (ids: number[]) => {
  return request.post('/api/v1/middlewares/batch-delete', { ids })
}

export const testMiddlewareConnection = (id: number) => {
  return request.post(`/api/v1/middlewares/${id}/test`)
}

export const executeMiddleware = (id: number, data: any) => {
  return request.post(`/api/v1/middlewares/${id}/execute`, data)
}

export const getMiddlewareDatabases = (id: number) => {
  return request.get(`/api/v1/middlewares/${id}/databases`)
}

export const getMiddlewareTables = (id: number, database: string) => {
  return request.get(`/api/v1/middlewares/${id}/tables`, { params: { database } })
}

export const getMiddlewareColumns = (id: number, database: string, table: string) => {
  return request.get(`/api/v1/middlewares/${id}/columns`, { params: { database, table } })
}

export const createDatabase = (id: number, name: string) => {
  return request.post(`/api/v1/middlewares/${id}/databases`, { name })
}

// ===== Redis 专用 API =====

export const getRedisInfo = (id: number, db?: number) => {
  return request.get(`/api/v1/middlewares/${id}/redis/info`, { params: { db } })
}

export const getRedisDatabases = (id: number) => {
  return request.get(`/api/v1/middlewares/${id}/redis/databases`)
}

export const scanRedisKeys = (id: number, params: { db?: number; cursor?: number; count?: number; pattern?: string }) => {
  return request.get(`/api/v1/middlewares/${id}/redis/keys`, { params })
}

export const getRedisKeyDetail = (id: number, key: string, db?: number) => {
  return request.get(`/api/v1/middlewares/${id}/redis/key`, { params: { key, db } })
}

export const setRedisKey = (id: number, data: { key: string; type: string; value: any; ttl?: number }, db?: number) => {
  return request.post(`/api/v1/middlewares/${id}/redis/key?db=${db ?? 0}`, data)
}

export const redisKeyAction = (id: number, data: { key: string; action: string; field?: string; value?: any; score?: number }, db?: number) => {
  return request.post(`/api/v1/middlewares/${id}/redis/key/action?db=${db ?? 0}`, data)
}

export const deleteRedisKeys = (id: number, keys: string[], db?: number) => {
  return request.delete(`/api/v1/middlewares/${id}/redis/key`, { data: { keys }, params: { db } })
}

export const setRedisKeyTTL = (id: number, data: { key: string; ttl: number }, db?: number) => {
  return request.put(`/api/v1/middlewares/${id}/redis/key/ttl?db=${db ?? 0}`, data)
}

export const renameRedisKey = (id: number, data: { oldKey: string; newKey: string }, db?: number) => {
  return request.put(`/api/v1/middlewares/${id}/redis/key/rename?db=${db ?? 0}`, data)
}

// ===== ClickHouse 专用 API =====

export const getClickHouseDatabases = (id: number) => {
  return request.get(`/api/v1/middlewares/${id}/clickhouse/databases`)
}

export const getClickHouseTables = (id: number, database: string) => {
  return request.get(`/api/v1/middlewares/${id}/clickhouse/tables`, { params: { database } })
}

export const getClickHouseColumns = (id: number, database: string, table: string) => {
  return request.get(`/api/v1/middlewares/${id}/clickhouse/columns`, { params: { database, table } })
}

export const createClickHouseDatabase = (id: number, name: string) => {
  return request.post(`/api/v1/middlewares/${id}/clickhouse/databases`, { name })
}

// ===== MongoDB 专用 API =====

export const getMongoDatabases = (id: number) => {
  return request.get(`/api/v1/middlewares/${id}/mongo/databases`)
}

export const getMongoCollections = (id: number, database: string) => {
  return request.get(`/api/v1/middlewares/${id}/mongo/collections`, { params: { database } })
}

export const createMongoCollection = (id: number, database: string, collection: string) => {
  return request.post(`/api/v1/middlewares/${id}/mongo/collections`, { database, collection })
}

export const queryMongoDocuments = (id: number, data: { database: string; collection: string; filter?: any; sort?: any; limit?: number; skip?: number }) => {
  return request.post(`/api/v1/middlewares/${id}/mongo/query`, data)
}

export const mongoInsertDocument = (id: number, data: { database: string; collection: string; document: any }) => {
  return request.post(`/api/v1/middlewares/${id}/mongo/insert`, data)
}

export const mongoUpdateDocuments = (id: number, data: { database: string; collection: string; filter: any; update: any }) => {
  return request.post(`/api/v1/middlewares/${id}/mongo/update`, data)
}

export const mongoDeleteDocuments = (id: number, data: { database: string; collection: string; filter: any }) => {
  return request.post(`/api/v1/middlewares/${id}/mongo/delete`, data)
}

export const getMongoStats = (id: number) => {
  return request.get(`/api/v1/middlewares/${id}/mongo/stats`)
}

export const getMongoCollectionStats = (id: number, database: string, collection: string) => {
  return request.get(`/api/v1/middlewares/${id}/mongo/collection-stats`, { params: { database, collection } })
}

// ===== 中间件权限管理 =====

export const getMiddlewarePermissions = (params: any) => {
  return request.get('/api/v1/middleware-permissions', { params })
}

export const createMiddlewarePermission = (data: any) => {
  return request.post('/api/v1/middleware-permissions', data)
}

export const updateMiddlewarePermission = (id: number, data: any) => {
  return request.put(`/api/v1/middleware-permissions/${id}`, data)
}

export const deleteMiddlewarePermission = (id: number) => {
  return request.delete(`/api/v1/middleware-permissions/${id}`)
}

// ===== Kafka 专用 API =====

export const getKafkaBrokers = (id: number) => {
  return request.get(`/api/v1/middlewares/${id}/kafka/brokers`)
}

export const getKafkaTopics = (id: number) => {
  return request.get(`/api/v1/middlewares/${id}/kafka/topics`)
}

export const createKafkaTopic = (id: number, data: { name: string; numPartitions?: number; replicationFactor?: number; config?: Record<string, string> }) => {
  return request.post(`/api/v1/middlewares/${id}/kafka/topics`, data)
}

export const deleteKafkaTopic = (id: number, topic: string) => {
  return request.delete(`/api/v1/middlewares/${id}/kafka/topics`, { params: { topic } })
}

export const getKafkaTopicDetail = (id: number, topic: string) => {
  return request.get(`/api/v1/middlewares/${id}/kafka/topic-detail`, { params: { topic } })
}

export const getKafkaTopicConfig = (id: number, topic: string) => {
  return request.get(`/api/v1/middlewares/${id}/kafka/topic-config`, { params: { topic } })
}

export const updateKafkaTopicConfig = (id: number, data: { topic: string; configs: Record<string, string> }) => {
  return request.put(`/api/v1/middlewares/${id}/kafka/topic-config`, data)
}

export const getKafkaConsumerGroups = (id: number) => {
  return request.get(`/api/v1/middlewares/${id}/kafka/consumer-groups`)
}

export const getKafkaConsumerGroupDetail = (id: number, groupId: string) => {
  return request.get(`/api/v1/middlewares/${id}/kafka/consumer-group-detail`, { params: { groupId } })
}

export const deleteKafkaConsumerGroup = (id: number, groupId: string) => {
  return request.delete(`/api/v1/middlewares/${id}/kafka/consumer-groups`, { params: { groupId } })
}

export const produceKafkaMessage = (id: number, data: { topic: string; key?: string; value: string; headers?: Record<string, string>; partition?: number }) => {
  return request.post(`/api/v1/middlewares/${id}/kafka/produce`, data)
}

export const startKafkaConsumerSession = (id: number, data: { topic: string; startOffset?: string }) => {
  return request.post(`/api/v1/middlewares/${id}/kafka/consumer-session/start`, data)
}

export const pollKafkaConsumerSession = (id: number, params: { sessionId: string; keyword?: string; limit?: number }) => {
  return request.get(`/api/v1/middlewares/${id}/kafka/consumer-session/poll`, { params })
}

export const stopKafkaConsumerSession = (id: number, sessionId: string) => {
  return request.delete(`/api/v1/middlewares/${id}/kafka/consumer-session/stop`, { params: { sessionId } })
}

// ===== Milvus 专用 API =====

export const getMilvusDatabases = (id: number) => {
  return request.get(`/api/v1/middlewares/${id}/milvus/databases`)
}

export const createMilvusDatabase = (id: number, name: string) => {
  return request.post(`/api/v1/middlewares/${id}/milvus/databases`, { name })
}

export const dropMilvusDatabase = (id: number, name: string) => {
  return request.delete(`/api/v1/middlewares/${id}/milvus/databases`, { params: { name } })
}

export const getMilvusCollections = (id: number, database?: string) => {
  return request.get(`/api/v1/middlewares/${id}/milvus/collections`, { params: { database } })
}

export const describeMilvusCollection = (id: number, collection: string, database?: string) => {
  return request.get(`/api/v1/middlewares/${id}/milvus/collection-detail`, { params: { collection, database } })
}

export const createMilvusCollection = (id: number, data: any) => {
  return request.post(`/api/v1/middlewares/${id}/milvus/collections`, data)
}

export const dropMilvusCollection = (id: number, collection: string, database?: string) => {
  return request.delete(`/api/v1/middlewares/${id}/milvus/collections`, { params: { collection, database } })
}

export const loadMilvusCollection = (id: number, data: { collection: string; database?: string }) => {
  return request.post(`/api/v1/middlewares/${id}/milvus/collection/load`, data)
}

export const releaseMilvusCollection = (id: number, data: { collection: string; database?: string }) => {
  return request.post(`/api/v1/middlewares/${id}/milvus/collection/release`, data)
}

export const createMilvusIndex = (id: number, data: any) => {
  return request.post(`/api/v1/middlewares/${id}/milvus/index`, data)
}

export const dropMilvusIndex = (id: number, collection: string, fieldName: string, database?: string) => {
  return request.delete(`/api/v1/middlewares/${id}/milvus/index`, { params: { collection, fieldName, database } })
}

export const queryMilvusData = (id: number, data: any) => {
  return request.post(`/api/v1/middlewares/${id}/milvus/query`, data)
}

export const insertMilvusData = (id: number, data: any) => {
  return request.post(`/api/v1/middlewares/${id}/milvus/insert`, data)
}

export const deleteMilvusData = (id: number, data: any) => {
  return request.post(`/api/v1/middlewares/${id}/milvus/delete`, data)
}

export const searchMilvusVectors = (id: number, data: any) => {
  return request.post(`/api/v1/middlewares/${id}/milvus/search`, data)
}

export const getMilvusPartitions = (id: number, collection: string, database?: string) => {
  return request.get(`/api/v1/middlewares/${id}/milvus/partitions`, { params: { collection, database } })
}

export const createMilvusPartition = (id: number, data: { collection: string; partition: string; database?: string }) => {
  return request.post(`/api/v1/middlewares/${id}/milvus/partitions`, data)
}

export const dropMilvusPartition = (id: number, collection: string, partition: string, database?: string) => {
  return request.delete(`/api/v1/middlewares/${id}/milvus/partitions`, { params: { collection, partition, database } })
}

export const getMilvusMetrics = (id: number) => {
  return request.get(`/api/v1/middlewares/${id}/milvus/metrics`)
}
