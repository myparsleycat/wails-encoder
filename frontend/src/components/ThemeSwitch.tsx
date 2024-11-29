import { useTheme } from 'next-themes'
import { SunIcon, MoonIcon } from 'lucide-react'
import { cn } from '@/lib/utils'

const ThemeChanger = ({ className }: { className?: string }) => {
  const { systemTheme, theme, setTheme } = useTheme()
  const currentTheme = theme === 'system' ? systemTheme : theme

  const toggleTheme = () => {
    setTheme(currentTheme === 'dark' ? 'light' : 'dark')
  }

  return (
    <button
      className={cn(
        className,
        'inline-flex items-center justify-center whitespace-nowrap text-sm font-medium ring-offset-background transition-colors',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
        'disabled:pointer-events-none disabled:opacity-50 border border-input hover:bg-accent',
        'hover:text-accent-foreground rounded-full w-10 h-10 bg-background'
      )}
      onClick={toggleTheme}
    >
      <MoonIcon className="w-5 h-5 rotate-90 scale-0 transition-transform ease-in-out duration-500 dark:rotate-0 dark:scale-100" />
      <SunIcon className="absolute w-5 h-5 rotate-0 scale-100 transition-transform ease-in-out duration-500 dark:-rotate-90 dark:scale-0" />
    </button>
  )
}

export default ThemeChanger