import type { Component, ComputedRef, Ref, WritableComputedRef } from 'vue'
import type { DatabaseDriver, SSLMode } from '@/lib/api/setup'

export type SetupStepId = 'admin' | 'database' | 'done'
export type TransitionDirection = 'forward' | 'backward'

export interface AdminDraft {
  username: string
  password: string
}

export interface PostgreSQLDraft {
  host: string
  port: number
  database: string
  username: string
  password: string
  sslMode: SSLMode
}

export interface SetupDatabaseDraft {
  driver: DatabaseDriver
  postgresql: PostgreSQLDraft
}

export interface SetupStepDefinition {
  id: SetupStepId
  icon: Component
  component: Component
  title: () => string
}

export interface SetupWizard {
  steps: readonly SetupStepDefinition[]
  currentStep: ComputedRef<SetupStepDefinition>
  stepperValue: WritableComputedRef<number>
  transitionClasses: ComputedRef<{
    enterActive: string
    enterFrom: string
    enterTo: string
    leaveActive: string
    leaveFrom: string
    leaveTo: string
  }>
  submitting: Ref<boolean>
  finalizationRequired: ComputedRef<boolean>
  finalizationDriver: ComputedRef<DatabaseDriver | null>
  autoLoginFailed: Ref<boolean>
  adminDraft: Ref<AdminDraft>
  databaseDraft: Ref<SetupDatabaseDraft>
  completeAdmin: (draft: AdminDraft) => void
  completeDatabase: () => Promise<void>
  back: () => void
  finish: () => Promise<void>
  isStepSelectable: (stepNumber: number) => boolean
}
