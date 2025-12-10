import { useEffect, useRef, forwardRef, useImperativeHandle } from 'react';
import cytoscape, { type Core } from 'cytoscape';
import cola from 'cytoscape-cola';
import { useForestStore } from '../../stores/forestStore';
import { useSettingsStore } from '../../stores/settingsStore';
import type { Node, Edge } from '../../types/graph';

cytoscape.use(cola);

interface ForestGraphProps {
  className?: string;
}

export interface ForestGraphHandle {
  resetLayout: () => void;
  zoomIn: () => void;
  zoomOut: () => void;
  fit: () => void;
}

export const ForestGraph = forwardRef<ForestGraphHandle, ForestGraphProps>(
  ({ className = '' }, ref) => {
    const containerRef = useRef<HTMLDivElement>(null);
    const cyRef = useRef<Core | null>(null);

    const nodes = useForestStore((state) => state.nodes);
    const edges = useForestStore((state) => state.edges);
    const setSelectedNode = useForestStore((state) => state.setSelectedNode);
    const setSelectedEdge = useForestStore((state) => state.setSelectedEdge);

    const forestMapSettings = useSettingsStore((state) => state.settings.forestMap);

    useImperativeHandle(ref, () => ({
      resetLayout: () => {
        if (!cyRef.current) return;
        const layout = cyRef.current.layout({
          name: forestMapSettings.layout,
          animate: forestMapSettings.animationDuration > 0,
          animationDuration: forestMapSettings.animationDuration,
          refresh: 1,
          maxSimulationTime: 2000,
          ungrabifyWhileSimulating: false,
          fit: true,
          padding: 30,
          nodeDimensionsIncludeLabels: true,
          randomize: false,
          avoidOverlap: true,
          handleDisconnected: true,
          convergenceThreshold: 0.01,
          nodeSpacing: forestMapSettings.nodeSpacing,
          edgeLength: forestMapSettings.edgeLength,
        });
        layout.run();
      },
      zoomIn: () => {
        if (!cyRef.current) return;
        const zoom = cyRef.current.zoom();
        cyRef.current.zoom({
          level: zoom * 1.2,
          renderedPosition: {
            x: cyRef.current.width() / 2,
            y: cyRef.current.height() / 2,
          },
        });
      },
      zoomOut: () => {
        if (!cyRef.current) return;
        const zoom = cyRef.current.zoom();
        cyRef.current.zoom({
          level: zoom * 0.8,
          renderedPosition: {
            x: cyRef.current.width() / 2,
            y: cyRef.current.height() / 2,
          },
        });
      },
      fit: () => {
        if (!cyRef.current) return;
        cyRef.current.fit(undefined, 30);
      },
    }));

    useEffect(() => {
    if (!containerRef.current) return;

    // If already initialized, skip (handles React Strict Mode double render)
    if (cyRef.current) return;

    const cy = cytoscape({
      container: containerRef.current,
      elements: [],
      style: [
        {
          selector: 'node',
          style: {
            'background-color': '#3b82f6',
            'label': (ele: any) => {
              if (!forestMapSettings.showLabels) return '';
              const label = ele.data('label');
              const ip = ele.data('properties')?.ip;
              return ip ? `${label}\n${ip}` : label;
            },
            'color': '#e2e8f0',
            'text-valign': 'bottom',
            'text-halign': 'center',
            'text-margin-y': 5,
            'font-size': '11px',
            'text-wrap': 'wrap',
            'text-max-width': '120px',
            'width': 40,
            'height': 40,
          },
        },
        {
          selector: 'node[type="system"]',
          style: {
            'background-color': '#8b5cf6',
            'shape': 'diamond',
            'width': 50,
            'height': 50,
          },
        },
        {
          selector: 'node[type="workstation"]',
          style: {
            'background-color': '#3b82f6',
            'shape': 'ellipse',
          },
        },
        {
          selector: 'node[type="server"]',
          style: {
            'background-color': '#10b981',
            'shape': 'round-rectangle',
          },
        },
        {
          selector: 'node[type="network"]',
          style: {
            'background-color': '#64748b',
            'shape': 'round-octagon',
            'width': 60,
            'height': 60,
          },
        },
        {
          selector: 'node[status="active"]',
          style: {
            'border-width': 3,
            'border-color': '#10b981',
          },
        },
        {
          selector: 'node[status="inactive"]',
          style: {
            'border-width': 3,
            'border-color': '#ef4444',
          },
        },
        {
          selector: 'node:selected',
          style: {
            'border-width': 4,
            'border-color': '#fbbf24',
            'background-color': '#60a5fa',
          },
        },
        {
          selector: 'edge',
          style: {
            'width': 2,
            'line-color': '#475569',
            'target-arrow-color': '#475569',
            'target-arrow-shape': 'triangle',
            'curve-style': 'bezier',
            'label': (ele: any) => forestMapSettings.showEdgeLabels ? ele.data('label') : '',
            'color': '#e2e8f0',
            'font-size': '9px',
            'text-rotation': 'autorotate',
            'text-margin-y': -10,
          },
        },
        {
          selector: 'edge[type="c2_connection"]',
          style: {
            'line-color': '#3b82f6',
            'target-arrow-color': '#3b82f6',
            'line-style': 'solid',
          },
        },
        {
          selector: 'edge[type="lateral_movement"]',
          style: {
            'line-color': '#f59e0b',
            'target-arrow-color': '#f59e0b',
            'line-style': 'dashed',
          },
        },
        {
          selector: 'edge[type="data_flow"]',
          style: {
            'line-color': '#06b6d4',
            'target-arrow-color': '#06b6d4',
            'line-style': 'dotted',
          },
        },
        {
          selector: 'edge:selected',
          style: {
            'line-color': '#fbbf24',
            'target-arrow-color': '#fbbf24',
            'width': 3,
          },
        },
      ],
      layout: {
        name: forestMapSettings.layout,
        animate: forestMapSettings.animationDuration > 0,
        animationDuration: forestMapSettings.animationDuration,
        refresh: 1,
        maxSimulationTime: 2000,
        ungrabifyWhileSimulating: false,
        fit: true,
        padding: 30,
        nodeDimensionsIncludeLabels: true,
        randomize: false,
        avoidOverlap: true,
        handleDisconnected: true,
        convergenceThreshold: 0.01,
        nodeSpacing: forestMapSettings.nodeSpacing,
        edgeLength: forestMapSettings.edgeLength,
      },
      minZoom: 0.3,
      maxZoom: 3,
    });

    const handleNodeTap = (event: any) => {
      const node = event.target;
      const nodeData: Node = {
        id: node.data('id'),
        type: node.data('type'),
        label: node.data('label'),
        status: node.data('status'),
        properties: node.data('properties') || {},
        createdAt: new Date(node.data('createdAt')),
        updatedAt: new Date(node.data('updatedAt')),
      };
      setSelectedNode(nodeData);
      setSelectedEdge(null);
    };

    const handleEdgeTap = (event: any) => {
      const edge = event.target;
      const edgeData: Edge = {
        id: edge.data('id'),
        source: edge.data('source'),
        target: edge.data('target'),
        type: edge.data('type'),
        label: edge.data('label'),
        properties: edge.data('properties') || {},
        createdAt: new Date(edge.data('createdAt')),
      };
      setSelectedEdge(edgeData);
      setSelectedNode(null);
    };

    const handleBackgroundTap = (event: any) => {
      if (event.target === cy) {
        setSelectedNode(null);
        setSelectedEdge(null);
      }
    };

    cy.on('tap', 'node', handleNodeTap);
    cy.on('tap', 'edge', handleEdgeTap);
    cy.on('tap', handleBackgroundTap);

    cyRef.current = cy;

    return () => {
      // Clear the container first to remove DOM event listeners
      if (containerRef.current) {
        containerRef.current.innerHTML = '';
      }

      if (cyRef.current) {
        try {
          // Destroy the cytoscape instance
          cyRef.current.destroy();
        } catch (e) {
          // Ignore errors during cleanup
        }
        cyRef.current = null;
      }
    };
  }, [setSelectedNode, setSelectedEdge]);

  useEffect(() => {
    if (!cyRef.current) return;

    const cy = cyRef.current;

    cy.elements().remove();

    const cyNodes = nodes.map((node) => ({
      data: {
        id: node.id,
        label: node.label,
        type: node.type,
        status: node.status,
        properties: node.properties,
        createdAt: node.createdAt.toISOString(),
        updatedAt: node.updatedAt.toISOString(),
      },
    }));

    const cyEdges = edges.map((edge) => ({
      data: {
        id: edge.id,
        source: edge.source,
        target: edge.target,
        label: edge.label,
        type: edge.type,
        properties: edge.properties,
        createdAt: edge.createdAt.toISOString(),
      },
    }));

    console.log('[ForestGraph] Adding edges to cytoscape:', cyEdges.map(e => `${e.data.id}: ${e.data.source} -> ${e.data.target}`));

    cy.add([...cyNodes, ...cyEdges]);

    const layout = cy.layout({
      name: forestMapSettings.layout,
      animate: forestMapSettings.animationDuration > 0,
      animationDuration: forestMapSettings.animationDuration,
      refresh: 1,
      maxSimulationTime: 2000,
      ungrabifyWhileSimulating: false,
      fit: forestMapSettings.autoCenter,
      padding: 30,
      nodeDimensionsIncludeLabels: true,
      randomize: false,
      avoidOverlap: true,
      handleDisconnected: true,
      convergenceThreshold: 0.01,
      nodeSpacing: forestMapSettings.nodeSpacing,
      edgeLength: forestMapSettings.edgeLength,
    });

    layout.run();
    }, [nodes, edges, forestMapSettings]);

  // Update styles when label settings change
  useEffect(() => {
    if (!cyRef.current) return;

    const cy = cyRef.current;

    // Update node label visibility
    cy.style()
      .selector('node')
      .style({
        'label': (ele: any) => {
          if (!forestMapSettings.showLabels) return '';
          const label = ele.data('label');
          const ip = ele.data('properties')?.ip;
          return ip ? `${label}\n${ip}` : label;
        },
      })
      .update();

    // Update edge label visibility
    cy.style()
      .selector('edge')
      .style({
        'label': (ele: any) => forestMapSettings.showEdgeLabels ? ele.data('label') : '',
      })
      .update();
  }, [forestMapSettings.showLabels, forestMapSettings.showEdgeLabels]);

    return (
      <div
        ref={containerRef}
        className={`w-full h-full bg-dark-bg-secondary rounded-lg ${className}`}
      />
    );
  }
);
