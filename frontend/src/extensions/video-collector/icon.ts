import { h } from 'vue'

export const VideoCollectorIcon = {
  render: () => h(
    'svg',
    { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
    [
      h('path', {
        'stroke-linecap': 'round',
        'stroke-linejoin': 'round',
        d: 'm15.75 10.5 4.72-2.36a.75.75 0 0 1 1.08.67v6.38a.75.75 0 0 1-1.08.67l-4.72-2.36m-10.5 4.5h8.25A2.25 2.25 0 0 0 15.75 15.75v-7.5A2.25 2.25 0 0 0 13.5 6H5.25A2.25 2.25 0 0 0 3 8.25v7.5A2.25 2.25 0 0 0 5.25 18Z'
      })
    ]
  )
}
