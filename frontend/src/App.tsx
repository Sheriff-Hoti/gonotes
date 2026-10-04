import { Shapeshift } from "@/components/shapeshift/Shapeshift";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import "./index.css";
import { SidebarProvider, SidebarTrigger } from "./components/ui/sidebar";
import { AppSidebar } from "./components/ui/app-sidebar";

function App() {
  return (
    <TooltipProvider>
      <SidebarProvider>
        <AppSidebar />
         <SidebarTrigger/>
      <Shapeshift />
        <Toaster />
      </SidebarProvider>
    </TooltipProvider>
  );
}

export default App;
