// Package bootstrap 提供依赖注入的引导和粘合层。
//
// 在分层架构中，理想的依赖关系应该是单向的 ( provider → dao → repo → service → api → app )。
//
// 但在某些业务场景下会出现双向依赖：
//   - 消息队列 ( mq )
//     需要调用 service 处理消息
//     同时 service 也需要通过 mq 发送消息
//   - 变更数据捕获 ( cdc )
//     需要调用 service 处理数据变更
//     同时 service 也会触发 CDC 事件
//
// 这种双向依赖会导致 Go 的循环导入错误 ( import cycle not allowed )。
//
// bootstrap 包作为依赖注入的 "粘合层" 位于依赖图的最上层：
//   - 可以同时导入有双向依赖的包 ( 如 mq、cdc、service )。
//   - 负责连接这些组件，配置它们之间的依赖关系。
//   - 不包含业务逻辑，只负责依赖注入的配置。
package bootstrap
