-- Kafka table engine for consuming events from Kafka
CREATE TABLE IF NOT EXISTS event_kafka
(
    `pid` UInt16 COMMENT 'Unique project identifier',
    `created_at` DateTime('UTC') COMMENT 'UTC dt',
    `deleted` UInt8,
    `gid` UInt64,
    `retention_days` UInt8,
    `event_id` UUID COMMENT 'Unique event identifier',
    `platform` UInt8 COMMENT 'Platform identifier Go, Python, etc.',
    `env` LowCardinality(String) COMMENT 'Environment identifier (dev, stage, prod, etc)',
    `release` LowCardinality(String) COMMENT 'App version in semver',
    `ipv4` IPv4 COMMENT 'Sender ip addr version 4',
    `ipv6` IPv6 COMMENT 'Sender ip addr version 6',
    `user` String,
    `user_email` String COMMENT 'User email',
    `user_name` String COMMENT 'User name',
    `user_username` String COMMENT 'User username',
    `sdk_id` UInt8 COMMENT 'SDK identifier',
    `sdk_version` LowCardinality(String) COMMENT 'SDK semver version',
    `tags.key` Array(String) COMMENT 'Tags key array',
    `tags.value` Array(String) COMMENT 'Tags value array',
    `contexts.key` Array(String) COMMENT 'Contexts key array',
    `contexts.value` Array(String) COMMENT 'Contexts value array',
    `primary_hash` UUID COMMENT 'Primary hash',
    `message` String COMMENT 'Message',
    `title` String COMMENT 'Title',
    `level` UInt8 COMMENT 'Log level',
    `type` UInt8 COMMENT 'Event type',
    `exception_stacks.type` Array(String) COMMENT 'Exception stack types',
    `exception_stacks.value` Array(String) COMMENT 'Exception stack values',
    `exception_frames.abs_path` Array(String) COMMENT 'Exception frame absolute path',
    `exception_frames.colno` Array(UInt32) COMMENT 'Exception frame column number',
    `exception_frames.filename` Array(String) COMMENT 'Exception frame filename',
    `exception_frames.function` Array(String) COMMENT 'Exception frame function',
    `exception_frames.lineno` Array(UInt32) COMMENT 'Exception frame line number',
    `exception_frames.in_app` Array(UInt8) COMMENT 'Exception frame in app'
)
ENGINE = Kafka
SETTINGS kafka_broker_list = 'redpanda:9092',
         kafka_topic_list = 'warnly.queue',
         kafka_group_name = 'clickhouse-event-reader-v2',
         kafka_format = 'JSONEachRow',
         kafka_num_consumers = 1,
         kafka_poll_timeout_ms = 1000,
         kafka_skip_broken_messages = 0,
         date_time_input_format = 'best_effort';

SET stream_like_engine_allow_direct_select=1;

-- Materialized view to consume from Kafka table and insert into main event table
CREATE MATERIALIZED VIEW IF NOT EXISTS event_kafka_mv TO event AS
SELECT
    *
FROM event_kafka SETTINGS stream_like_engine_allow_direct_select=1;
