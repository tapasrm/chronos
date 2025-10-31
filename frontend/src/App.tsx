import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import CronJobManager from './components/CronJobManager';
import Header from './components/Header';

const queryClient = new QueryClient();

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
  <Header />
  <CronJobManager />
    </QueryClientProvider>
  );
}
