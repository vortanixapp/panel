type ContentSectionProps = {
  title: string
  desc: string
  children: React.JSX.Element
}

export function ContentSection({ title, desc, children }: ContentSectionProps) {
  return (
    <div className="flex flex-col gap-5">
      <div className="space-y-1.5">
        <h2 className="text-[19px] leading-none font-semibold">{title}</h2>
        <p className="text-[13px] text-muted-foreground">{desc}</p>
      </div>
      {children}
    </div>
  )
}
