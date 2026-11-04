<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  Stepper,
  StepperIndicator,
  StepperItem,
  StepperSeparator,
  StepperTitle,
  StepperTrigger,
} from '@/components/ui/stepper'
import { useSetupWizard } from '@/composables/setup/useSetupWizard'

const { t } = useI18n()
const wizard = useSetupWizard()
const {
  steps,
  currentStep,
  stepperValue,
  transitionClasses,
  isStepSelectable,
} = wizard
const totalSteps = steps.length
</script>

<template>
  <div class="bg-background px-6 py-10 text-foreground min-block-screen">
    <div
      class="mx-auto flex items-center inline-full max-inline-2xl min-block-[calc(100vh-5rem)]"
    >
      <div class="space-y-8 inline-full">
        <div class="space-y-2 text-center">
          <h1 class="text-3xl font-semibold tracking-tight sm:text-4xl">
            {{ t('setup.title') }}
          </h1>
        </div>

        <Stepper
          v-model="stepperValue"
          :linear="false"
          class="flex items-start gap-3 inline-full"
        >
          <StepperItem
            v-for="(step, index) in steps"
            :key="step.id"
            :step="index + 1"
            :disabled="!isStepSelectable(index + 1)"
            class="relative flex flex-1 flex-col items-center justify-center gap-2 text-center transition-opacity data-disabled:pointer-events-none data-disabled:opacity-50"
          >
            <StepperSeparator
              v-if="index + 1 !== totalSteps"
              class="absolute inset-s-[calc(50%+1.5rem)] inset-e-[calc(-50%+1.5rem)] inset-bs-4 block rounded-full bg-border/70 block-px"
            />

            <StepperTrigger as-child>
              <StepperIndicator class="shadow-sm block-10 inline-10">
                <component
                  :is="step.icon"
                  class="block-4 inline-4"
                />
              </StepperIndicator>
            </StepperTrigger>

            <StepperTitle class="text-sm font-medium text-foreground">
              {{ step.title() }}
            </StepperTitle>
          </StepperItem>
        </Stepper>

        <div
          class="relative overflow-hidden rounded-2xl border border-border/60 bg-card/70 p-6 shadow-sm sm:p-8"
        >
          <Transition
            mode="out-in"
            :enter-active-class="transitionClasses.enterActive"
            :enter-from-class="transitionClasses.enterFrom"
            :enter-to-class="transitionClasses.enterTo"
            :leave-active-class="transitionClasses.leaveActive"
            :leave-from-class="transitionClasses.leaveFrom"
            :leave-to-class="transitionClasses.leaveTo"
          >
            <section
              :key="currentStep.id"
              class="flex flex-col gap-6 min-block-96"
            >
              <component
                :is="currentStep.component"
                :wizard="wizard"
              />
            </section>
          </Transition>
        </div>
      </div>
    </div>
  </div>
</template>
