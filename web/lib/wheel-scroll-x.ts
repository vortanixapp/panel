export function wheelScrollX(node: HTMLElement | null) {
  if (!node) return;

  const onWheel = (event: WheelEvent) => {
    if (event.ctrlKey || event.defaultPrevented) return;
    if (Math.abs(event.deltaY) <= Math.abs(event.deltaX)) return;
    if (node.scrollWidth <= node.clientWidth) return;
    const max = node.scrollWidth - node.clientWidth;
    const next = Math.min(max, Math.max(0, node.scrollLeft + event.deltaY));
    if (next === node.scrollLeft) return;
    node.scrollLeft = next;
    event.preventDefault();
  };

  node.addEventListener("wheel", onWheel, { passive: false });
  return () => node.removeEventListener("wheel", onWheel);
}
