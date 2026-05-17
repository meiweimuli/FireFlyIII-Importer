<template>
  <div class="rule-display">
    <div v-if="group.conditionLogic" class="logic-tag"><span v-if="group.negate" class="negate">NOT </span>{{ group.conditionLogic }}</div>
    <div class="conditions-list">
      <div v-for="(child, index) in group.conditions" :key="index" class="condition-item">
        <template v-if="child.isGroup">
          <RuleDisplay :group="child" />
        </template>
        <template v-else>
          <span v-if="child.negate" class="negate">NOT </span>
          <span class="field">{{ child.matchField }}</span>
          <span class="op">{{ child.matchType }}</span>
          <span class="value">"{{ child.matchValue }}"</span>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  group: any
}>()
</script>

<style scoped>
.rule-display {
  margin-top: 4px;
}
.logic-tag {
  font-size: 11px;
  color: #e6a23c;
  font-weight: bold;
  margin-bottom: 2px;
}
.conditions-list {
  padding-left: 12px;
  border-left: 2px solid #ebeef5;
}
.condition-item {
  margin-bottom: 4px;
  font-size: 13px;
}
.field {
  color: #409eff;
  font-weight: bold;
}
.op {
  color: #909399;
  margin: 0 4px;
  font-size: 12px;
}
.value {
  color: #67c23a;
}
.negate {
  color: #f56c6c;
  font-weight: bold;
}
</style>
