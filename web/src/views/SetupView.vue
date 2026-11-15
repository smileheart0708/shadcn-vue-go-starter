<script setup lang="ts">
import { useTemplateRef } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Stepper,
  StepperIndicator,
  StepperItem,
  StepperSeparator,
  StepperTitle,
  StepperTrigger,
} from '@/components/ui/stepper'
import {
  STEP_ENTER_DURATION_MS,
  useSetupWizard,
} from '@/composables/setup/useSetupWizard'

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

const cardRef = useTemplateRef<HTMLDivElement>('card')
let heightAnimation: Animation | undefined
let leavingCardHeight = 0

// 卡片高度由当前步骤的内容决定，而整列是垂直居中的：新内容一上来就把卡片换成新高度，
// 卡片连同 logo、Stepper 会在同一帧跳位，这才是「闪烁」的主因。离场时先记下旧高度，
// 入场时用 WAAPI 从旧高度补间到新高度——终态仍是布局高度，不需要定时器还原。
function captureCardHeight() {
  leavingCardHeight = cardRef.value?.getBoundingClientRect().height ?? 0
}

// 必须保持 0 个参数：Vue 见到带 done 回调的 enter 钩子就认为它自己收尾（hasExplicitCallback），
// 会跳过 transitionend 收尾，out-in 的新内容会永久停在 enter-from 的 opacity-0 上。
function animateCardHeight() {
  const card = cardRef.value
  if (
    card === null ||
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  ) {
    return
  }

  // 上一段补间还在跑时，rect 读到的是当前视觉高度（接上它才不跳），而它同时会把高度拖离布局值，
  // 所以必须先撤销再量终值：否则终值取到动画中间值，卡片缩到装不下内容，播完再跳回布局高度。
  const resuming = heightAnimation?.playState === 'running'
  const from = resuming
    ? card.getBoundingClientRect().height
    : leavingCardHeight
  heightAnimation?.cancel()
  heightAnimation = undefined

  const to = card.getBoundingClientRect().height
  if (from === 0 || from === to) {
    return
  }

  heightAnimation = card.animate(
    [{ height: `${String(from)}px` }, { height: `${String(to)}px` }],
    {
      duration: STEP_ENTER_DURATION_MS,
      easing: 'cubic-bezier(0.4,0,0.2,1)',
    },
  )
}
</script>

<template>
  <div class="bg-background px-6 py-10 text-foreground min-block-screen">
    <div
      class="mx-auto flex items-center inline-full max-inline-xl min-block-[calc(100vh-5rem)]"
    >
      <div class="space-y-6 inline-full">
        <div class="flex flex-col items-center gap-2">
          <img
            src="/logo.svg"
            :alt="t('app.logoAlt')"
            class="block-8 inline-8"
          />
          <p class="text-lg font-semibold tracking-tight">
            {{ t('app.name') }}
          </p>
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
          ref="card"
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
            @before-leave="captureCardHeight"
            @enter="animateCardHeight"
          >
            <!-- 高度必须由内容决定：这里再加 min-block-* 会让高度补间的起止值相同，动画直接失效。 -->
            <section
              :key="currentStep.id"
              class="flex flex-col gap-6"
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
