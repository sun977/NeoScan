# 核心组件 - 调度器（Scheduler）

- ScheduleManager ( scheduler/engine.go ): 负责定时触发和项目级流程控制。
- StageTransitionEngine (集成在 Scheduler 中): 负责 Stage 状态流转。