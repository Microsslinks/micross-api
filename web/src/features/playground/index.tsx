/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { PlaygroundChat } from './components/chat/playground-chat'
import { PlaygroundHeader } from './components/header/playground-header'
import { PlaygroundInput } from './components/input/playground-input'
import {
  useChatHandler,
  usePlaygroundConversation,
  usePlaygroundOptions,
  usePlaygroundState,
} from './hooks'

export function Playground() {
  const {
    config,
    parameterEnabled,
    messages,
    isLoadingMessages,
    models,
    groups,
    updateMessages,
    setModels,
    setGroups,
    updateConfig,
    updateParameterEnabled,
    clearMessages,
  } = usePlaygroundState()

  const { sendChat, stopGeneration, isGenerating } = useChatHandler({
    config,
    parameterEnabled,
    onMessageUpdate: updateMessages,
  })

  const {
    editingMessageKey,
    handleSendMessage,
    handleRegenerateMessage,
    handleEditMessage,
    handleEditOpenChange,
    applyEdit,
    handleDeleteMessage,
  } = usePlaygroundConversation({
    messages,
    updateMessages,
    sendChat,
  })

  const handleClearMessages = () => {
    handleEditOpenChange(false)
    clearMessages()
  }

  const { isLoadingModels } = usePlaygroundOptions({
    currentGroup: config.group,
    currentModel: config.model,
    setGroups,
    setModels,
    updateConfig,
  })

  return (
    <div className='relative flex size-full min-h-0 flex-col overflow-hidden'>
      {/* Subtle radial gradient backdrop — primary tint, follow theme */}
      <div
        aria-hidden='true'
        className='from-primary/5 via-transparent to-primary/3 pointer-events-none absolute inset-0 bg-gradient-to-br'
      />
      <div
        aria-hidden='true'
        className='from-primary/4 pointer-events-none absolute top-0 left-1/2 h-[420px] w-[860px] -translate-x-1/2 rounded-full bg-[radial-gradient(ellipse_at_center,var(--tw-gradient-stops))] via-transparent to-transparent blur-3xl'
      />

      <PlaygroundHeader
        models={models}
        modelValue={config.model}
        onModelChange={(value) => updateConfig('model', value)}
        isModelLoading={isLoadingModels}
        groups={groups}
        groupValue={config.group}
        onGroupChange={(value) => updateConfig('group', value)}
        hasMessages={messages.length > 0}
        onClearMessages={handleClearMessages}
        isGenerating={isGenerating}
      />

      {/* Full-width scroll container: scrolling works even over side whitespace */}
      <div className='flex min-h-0 flex-1 flex-col overflow-hidden'>
        <PlaygroundChat
          messages={messages}
          isLoadingMessages={isLoadingMessages}
          onRegenerateMessage={handleRegenerateMessage}
          onEditMessage={handleEditMessage}
          onDeleteMessage={handleDeleteMessage}
          onSelectPrompt={handleSendMessage}
          isGenerating={isGenerating}
          editingKey={editingMessageKey}
          onCancelEdit={handleEditOpenChange}
          onSaveEdit={(newContent) => applyEdit(newContent, false)}
          onSaveEditAndSubmit={(newContent) => applyEdit(newContent, true)}
        />
      </div>

      {/* Input area: center content and constrain to the same container width */}
      <div className='relative mx-auto w-full max-w-4xl'>
        <PlaygroundInput
          config={config}
          disabled={isGenerating}
          isGenerating={isGenerating}
          models={models}
          onConfigChange={updateConfig}
          onParameterEnabledChange={updateParameterEnabled}
          onStop={stopGeneration}
          onSubmit={handleSendMessage}
          parameterEnabled={parameterEnabled}
        />
      </div>
    </div>
  )
}