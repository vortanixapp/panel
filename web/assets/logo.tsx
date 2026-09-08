import { type SVGProps } from 'react'
import { cn } from '@/lib/utils'

export function Logo({ className, ...props }: SVGProps<SVGSVGElement>) {
  return (
    <svg
      id='vortanix-logo'
      viewBox='0 0 1006 1006'
      xmlns='http://www.w3.org/2000/svg'
      height='24'
      width='24'
      className={cn('size-6', className)}
      {...props}
    >
      <title>VORTANIX</title>
      <image
        href='/branding/vortanix-mark-square.png'
        width='1006'
        height='1006'
      />
    </svg>
  )
}
