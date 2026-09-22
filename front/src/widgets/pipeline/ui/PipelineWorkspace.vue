<script setup lang="ts">
import ErrorState from '@/shared/ui/ErrorState.vue'
import { usePipelineWorkspace } from '../model/usePipelineWorkspace'
import PipelineBoardHeader from './PipelineBoardHeader.vue'
import PipelineColumns from './PipelineColumns.vue'
import PipelineFragmentEditorDialog from './PipelineFragmentEditorDialog.vue'
import PipelineInstagramDialog from './PipelineInstagramDialog.vue'
import PipelineRenderDialog from './PipelineRenderDialog.vue'
import PipelineVideoPreviewDialog from './PipelineVideoPreviewDialog.vue'

const workspace = usePipelineWorkspace()
const { error } = workspace
</script>

<template>
  <section>
    <ErrorState v-if="error" class="mt-4" :message="error" />
    <PipelineBoardHeader :workspace="workspace" />
    <PipelineColumns :workspace="workspace" />
    <PipelineVideoPreviewDialog :workspace="workspace" />
    <PipelineInstagramDialog :workspace="workspace" />
    <PipelineFragmentEditorDialog :workspace="workspace" />
    <PipelineRenderDialog :workspace="workspace" />
  </section>
</template>
