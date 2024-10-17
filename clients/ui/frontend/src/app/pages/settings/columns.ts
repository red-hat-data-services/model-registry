import { SortableData } from '~/app/components/table';
import { ModelRegistry } from '~/app/types';

export const modelRegistryColumns: SortableData<ModelRegistry>[] = [
  {
    field: 'model regisry name',
    label: 'Model registry name',
    sortable: (a, b) => a.name.localeCompare(b.name),
    width: 30,
  },
  // TODO: Add once we manage permissions
  // {
  //   field: 'status',
  //   label: 'Status',
  //   sortable: false,
  // },
  // {
  //   field: 'manage permissions',
  //   label: '',
  //   sortable: false,
  // },
  // kebabTableColumn(),
];
