import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ScenarioStateTimeline from './ScenarioStateTimeline.vue'

describe('ScenarioStateTimeline', () => {
  it('marks rework without pretending the scenario was accepted', () => {
    const wrapper = mount(ScenarioStateTimeline, { props: { state: 'rework' } })
    expect(wrapper.text()).toContain('退回修订')
    expect(wrapper.findAll('.scenario-stage.active')).toHaveLength(0)
  })

  it('shows the verified stage as active', () => {
    const wrapper = mount(ScenarioStateTimeline, { props: { state: 'verified' } })
    expect(wrapper.find('.scenario-stage.active').text()).toContain('已复核')
  })
})
