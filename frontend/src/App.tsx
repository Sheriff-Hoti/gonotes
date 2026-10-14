import { Shapeshift } from "@/components/shapeshift/Shapeshift";
import { ModeToggle } from "@/components/mode-toggle";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { ThemeProvider } from "next-themes";
import "./index.css";
import { SidebarProvider, SidebarTrigger } from "./components/ui/sidebar";
import { AppSidebar } from "./components/ui/app-sidebar";

function App() {
  return (
    <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
      <TooltipProvider>
        <SidebarProvider>
          <AppSidebar />
          <SidebarTrigger />
          <div className="fixed top-4 right-4 z-10">
            <ModeToggle />
          </div>
          <Shapeshift />
          <Toaster />
        </SidebarProvider>
      </TooltipProvider>
    </ThemeProvider>
  );
}

export default App;
