import { create } from 'zustand';
import type { Node, Edge, GraphData } from '../types/graph';
import { api } from '../services/api';

interface ForestState {
  nodes: Node[];
  edges: Edge[];
  selectedNode: Node | null;
  selectedEdge: Edge | null;
  isLoading: boolean;
  error: string | null;

  // API methods
  fetchGraphData: () => Promise<void>;
  refreshGraphData: () => Promise<void>;
  setSelectedNode: (node: Node | null) => void;
  setSelectedEdge: (edge: Edge | null) => void;
  updateGraph: (data: GraphData) => void;
  addNode: (node: Node) => void;
  addEdge: (edge: Edge) => void;
  removeNode: (nodeId: string) => void;
  removeEdge: (edgeId: string) => void;
  clearError: () => void;
}

export const useForestStore = create<ForestState>((set) => ({
  nodes: [],
  edges: [],
  selectedNode: null,
  selectedEdge: null,
  isLoading: false,
  error: null,

  fetchGraphData: async () => {
    set({ isLoading: true, error: null });

    try {
      const graphData = await api.forest.getGraphData();
      set({
        nodes: graphData.nodes,
        edges: graphData.edges,
        isLoading: false,
      });
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : 'Failed to fetch graph data';
      set({
        isLoading: false,
        error: errorMessage,
      });
      console.error('Failed to fetch graph data:', error);
    }
  },

  refreshGraphData: async () => {
    try {
      const graphData = await api.forest.refreshGraphData();
      set({
        nodes: graphData.nodes,
        edges: graphData.edges,
      });
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : 'Failed to refresh graph data';
      set({ error: errorMessage });
      console.error('Failed to refresh graph data:', error);
    }
  },

  setSelectedNode: (node) => set({ selectedNode: node }),

  setSelectedEdge: (edge) => set({ selectedEdge: edge }),

  updateGraph: (data) => set({ nodes: data.nodes, edges: data.edges }),

  addNode: (node) => set((state) => ({ nodes: [...state.nodes, node] })),

  addEdge: (edge) => set((state) => ({ edges: [...state.edges, edge] })),

  removeNode: (nodeId) =>
    set((state) => ({
      nodes: state.nodes.filter((n) => n.id !== nodeId),
      edges: state.edges.filter(
        (e) => e.source !== nodeId && e.target !== nodeId
      ),
      selectedNode:
        state.selectedNode?.id === nodeId ? null : state.selectedNode,
    })),

  removeEdge: (edgeId) =>
    set((state) => ({
      edges: state.edges.filter((e) => e.id !== edgeId),
      selectedEdge: state.selectedEdge?.id === edgeId ? null : state.selectedEdge,
    })),

  clearError: () => set({ error: null }),
}));
