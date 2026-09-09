import { cn } from '@/lib/utils'

type MainProps = React.HTMLAttributes<HTMLElement> & {
  fixed?: boolean
  fluid?: boolean
  ref?: React.Ref<HTMLElement>
}

export function Main({ fixed, className, fluid = true, ...props }: MainProps) {
  return (
    <main
      data-layout={fixed ? 'fixed' : 'auto'}
      className={cn(
        'w-full min-w-0 px-4 py-6',

        fixed && 'flex min-h-0 flex-1 flex-col overflow-y-auto',

        !fluid &&
          '@7xl/content:mx-auto @7xl/content:w-full @7xl/content:max-w-7xl',
        className
      )}
      {...props}
    />
  )
}
