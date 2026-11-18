const REDUCED_MOTION_MEDIA_QUERY = '(prefers-reduced-motion: reduce)'

/*
 * 系统的「减少动效」偏好必须在启动动画之前判断，不能只靠 CSS 把时长压到 0：
 * View Transition 的全屏快照交叉淡入由浏览器伪元素驱动，页面样式管不到
 * `::view-transition-*`，只能选择不启动过渡。
 */
export function prefersReducedMotion(): boolean {
  return window.matchMedia(REDUCED_MOTION_MEDIA_QUERY).matches
}
