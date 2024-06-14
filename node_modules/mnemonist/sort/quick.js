/**
 * Mnemonist Quick Sort
 * =====================
 *
 * Quick sort related functions.
 * Adapted from: https://alienryderflex.com/quicksort/
 */
var LOS = new Float64Array(64),
    HIS = new Float64Array(64);

function inplaceQuickSort(array, lo, hi) {
  var p, i, l, r, swap;

  LOS[0] = lo;
  HIS[0] = hi;
  i = 0;

  while (i >= 0) {
    l = LOS[i];
    r = HIS[i] - 1;

    if (l < r) {
      p = array[l];

      while (l < r) {
        while (array[r] >= p && l < r)
          r--;

        if (l < r)
          array[l++] = array[r];

        while (array[l] <= p && l < r)
          l++;

        if (l < r)
          array[r--] = array[l];
      }

      array[l] = p;
      LOS[i + 1] = l + 1;
      HIS[i + 1] = HIS[i];
      HIS[i++] = l;

      if (HIS[i] - LOS[i] > HIS[i - 1] - LOS[i - 1]) {
        swap = LOS[i];
        LOS[i] = LOS[i - 1];
        LOS[i - 1] = swap;

        swap = HIS[i];
        HIS[i] = HIS[i - 1];
        HIS[i - 1] = swap;
      }
    }
    else {
      i--;
    }
  }

  return array;
}

exports.inplaceQuickSort = inplaceQuickSort;

function inplaceQuickSortIndices(array, indices, lo, hi) {
  var p, i, l, r, t, swap;

  LOS[0] = lo;
  HIS[0] = hi;
  i = 0;

  while (i >= 0) {
    l = LOS[i];
    r = HIS[i] - 1;

    if (l < r) {
      t = indices[l];
      p = array[t];

      while (l < r) {
        while (array[indices[r]] >= p && l < r)
          r--;

        if (l < r)
          indices[l++] = indices[r];

        while (array[indices[l]] <= p && l < r)
          l++;

        if (l < r)
          indices[r--] = indices[l];
      }

      indices[l] = t;
      LOS[i + 1] = l + 1;
      HIS[i + 1] = HIS[i];
      HIS[i++] = l;

      if (HIS[i] - LOS[i] > HIS[i - 1] - LOS[i - 1]) {
        swap = LOS[i];
        LOS[i] = LOS[i - 1];
        LOS[i - 1] = swap;

        swap = HIS[i];
        HIS[i] = HIS[i - 1];
        HIS[i - 1] = swap;
      }
    }
    else {
      i--;
    }
  }

  return indices;
}

exports.inplaceQuickSortIndices = inplaceQuickSortIndices;
