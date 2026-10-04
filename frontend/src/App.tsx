import { Shapeshift } from "@/components/shapeshift/Shapeshift";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import "./index.css";

function App() {
  return (
    <TooltipProvider>
      <Shapeshift />
      <Toaster />
    </TooltipProvider>
  );
}

export default App;
